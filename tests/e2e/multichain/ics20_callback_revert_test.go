package multichain_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/btcsuite/btcd/btcutil/bech32"
	transfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channel "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/tests/e2e/multichain"
)

const ics20CallbackRevertABI = `[
	{"type":"function","name":"transfer","stateMutability":"nonpayable","inputs":[
		{"name":"sourcePort","type":"string"},
		{"name":"sourceChannel","type":"string"},
		{"name":"denom","type":"string"},
		{"name":"amount","type":"uint256"},
		{"name":"sender","type":"address"},
		{"name":"receiver","type":"string"},
		{"name":"timeoutHeight","type":"tuple","components":[
			{"name":"revisionNumber","type":"uint64"},
			{"name":"revisionHeight","type":"uint64"}]},
		{"name":"timeoutTimestamp","type":"uint64"},
		{"name":"memo","type":"string"}],
	"outputs":[{"name":"nextSequence","type":"uint64"}]}
]`

// TestICS20CallbackRevertDoesNotEscapeEVMFrame proves that a successful nested
// SDK CacheContext write cannot escape a subsequently reverted EVM child frame.
//
// This is intentionally a safety-invariant test: it is RED with the vulnerable
// EVM dependency because escrow, packet state, and native supply survive the
// child-frame revert. It should become GREEN by changing only the dependency to
// an EVM version that preserves the outer snapshot boundary.
func TestICS20CallbackRevertDoesNotEscapeEVMFrame(t *testing.T) {
	if testing.Short() {
		t.Skip("requires dockerized XPLA and simd chains")
	}

	ctx := context.Background()
	setup := multichain.StartXplaChainAndSimdWithIBC(t, ctx, multichain.LocalImage)
	xplaChain := setup.XplaChain

	channels, err := setup.GetXplaChannels(ctx)
	require.NoError(t, err)
	require.Len(t, channels, 1)
	transferChannel := channels[0].ChannelID
	require.NoError(t, setup.StopRelayer(ctx), "the relayer must not consume any escaped packet")

	endpoint, err := xplaChain.GetFullNode().GetHostAddress(ctx, "8545/tcp")
	require.NoError(t, err)
	evm, err := ethclient.DialContext(ctx, endpoint)
	require.NoError(t, err)
	defer evm.Close()

	attacker := setup.XplaUsers[0]
	key, err := multichain.ExportEthWalletKey(ctx, xplaChain, attacker)
	require.NoError(t, err)

	chainID, err := evm.ChainID(ctx)
	require.NoError(t, err)
	gasPrice := big.NewInt(300_000_000_000)
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	require.NoError(t, err)
	auth.Context = ctx
	auth.GasLimit = 8_000_000
	auth.GasPrice = gasPrice

	artifactPath := "../../solidity/suites/precompiles/artifacts/contracts/test/ICS20CallbackRevert.sol/ICS20CallbackRevert.json"
	artifactJSON, err := os.ReadFile(artifactPath)
	require.NoError(t, err, "read %s; run make test-contracts-compile from the repository root", artifactPath)
	var artifact multichain.ContractArtifact
	require.NoError(t, json.Unmarshal(artifactJSON, &artifact))
	contractABI, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	require.NoError(t, err)
	contractAddress, contract, err := multichain.DeployEVMContract(ctx, evm, auth, artifact)
	require.NoError(t, err)

	contractBech32, err := xplaBech32Address(contractAddress)
	require.NoError(t, err)
	parsedICS20ABI, err := abi.JSON(strings.NewReader(ics20CallbackRevertABI))
	require.NoError(t, err)
	type heightTuple struct {
		RevisionNumber uint64 `abi:"revisionNumber"`
		RevisionHeight uint64 `abi:"revisionHeight"`
	}

	amount := big.NewInt(1_000_000_000_000_000_000)
	memo := fmt.Sprintf(`{"src_callback":{"address":"%s"}}`, contractBech32)
	calldata, err := parsedICS20ABI.Pack(
		"transfer",
		"transfer",
		transferChannel,
		multichain.Denom,
		amount,
		contractAddress,
		"undeliverable-recipient",
		heightTuple{},
		uint64(time.Now().Add(10*time.Minute).UnixNano()),
		memo,
	)
	require.NoError(t, err)

	escrowAddress := common.BytesToAddress(transfer.GetEscrowAddress("transfer", transferChannel).Bytes())
	escrowBefore, err := evm.BalanceAt(ctx, escrowAddress, nil)
	require.NoError(t, err)
	supplyBefore, err := xplaChain.BankQueryTotalSupplyOf(ctx, multichain.Denom)
	require.NoError(t, err)

	channelQuery := channel.NewQueryClient(xplaChain.GetFullNode().GrpcConn)
	sequenceBefore, err := channelQuery.NextSequenceSend(ctx, &channel.QueryNextSequenceSendRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)
	commitmentsBefore, err := channelQuery.PacketCommitments(ctx, &channel.QueryPacketCommitmentsRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)
	require.Empty(t, commitmentsBefore.Commitments)

	auth.GasLimit = 20_000_000
	auth.Value = amount
	attack, err := contract.Transact(auth, "attack", calldata)
	require.NoError(t, err)
	receipt, err := multichain.WaitForSuccessfulEVMReceipt(ctx, evm, attack)
	require.NoError(t, err, "the outer frame deliberately ignores the reverted child call")
	attackPacketSequence := requireICS20SuccessEvent(t, receipt, contractAddress, contractABI, "ChildICS20Succeeded")
	require.Equal(t, sequenceBefore.NextSequenceSend, attackPacketSequence,
		"the child must reach a successful ICS20 call before the deliberate revert")
	t.Logf("callback-revert transaction=%s block=%d gasUsed=%d", attack.Hash(), receipt.BlockNumber.Uint64(), receipt.GasUsed)

	escrowAfter, err := evm.BalanceAt(ctx, escrowAddress, nil)
	require.NoError(t, err)
	supplyAfter, err := xplaChain.BankQueryTotalSupplyOf(ctx, multichain.Denom)
	require.NoError(t, err)
	contractBalance, err := evm.BalanceAt(ctx, contractAddress, nil)
	require.NoError(t, err)
	sequenceAfter, err := channelQuery.NextSequenceSend(ctx, &channel.QueryNextSequenceSendRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)
	commitmentsAfter, err := channelQuery.PacketCommitments(ctx, &channel.QueryPacketCommitmentsRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)

	escrowDelta := new(big.Int).Sub(escrowAfter, escrowBefore)
	supplyDelta := new(big.Int).Sub(supplyAfter.Amount.BigInt(), supplyBefore.Amount.BigInt())
	t.Logf(
		"post-revert deltas: escrow=%s supply=%s nextSequence=%d->%d commitments=%d",
		escrowDelta,
		supplyDelta,
		sequenceBefore.NextSequenceSend,
		sequenceAfter.NextSequenceSend,
		len(commitmentsAfter.Commitments),
	)

	assert.Zero(t, escrowDelta.Sign(), "reverted child frame must not move native coins into IBC escrow")
	assert.Zero(t, supplyDelta.Sign(), "reverted child frame must not increase native supply")
	assert.Equal(t, sequenceBefore.NextSequenceSend, sequenceAfter.NextSequenceSend,
		"reverted child frame must not advance the packet sequence")
	assert.Empty(t, commitmentsAfter.Commitments,
		"reverted child frame must not leave packet commitments")
	assert.Zero(t, contractBalance.Cmp(amount),
		"the successful outer frame must retain only the value originally sent to the contract")

	// Execute the same precompile normally on the same patched chain. This
	// control proves that zero deltas above are caused by rollback, not by a
	// missing/disabled precompile or an early setup failure.
	controlCalldata, err := parsedICS20ABI.Pack(
		"transfer",
		"transfer",
		transferChannel,
		multichain.Denom,
		amount,
		contractAddress,
		setup.SimdUsers[0].FormattedAddress(),
		heightTuple{},
		uint64(time.Now().Add(10*time.Minute).UnixNano()),
		"",
	)
	require.NoError(t, err)
	auth.Value = nil
	auth.GasLimit = 12_000_000
	control, err := contract.Transact(auth, "controlTransfer", controlCalldata)
	require.NoError(t, err)
	controlReceipt, err := multichain.WaitForSuccessfulEVMReceipt(ctx, evm, control)
	require.NoError(t, err)
	controlPacketSequence := requireICS20SuccessEvent(t, controlReceipt, contractAddress, contractABI, "ControlICS20Succeeded")
	require.Equal(t, sequenceAfter.NextSequenceSend, controlPacketSequence)

	controlEscrow, err := evm.BalanceAt(ctx, escrowAddress, nil)
	require.NoError(t, err)
	controlSupply, err := xplaChain.BankQueryTotalSupplyOf(ctx, multichain.Denom)
	require.NoError(t, err)
	controlContractBalance, err := evm.BalanceAt(ctx, contractAddress, nil)
	require.NoError(t, err)
	controlSequence, err := channelQuery.NextSequenceSend(ctx, &channel.QueryNextSequenceSendRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)
	controlCommitments, err := channelQuery.PacketCommitments(ctx, &channel.QueryPacketCommitmentsRequest{
		PortId:    "transfer",
		ChannelId: transferChannel,
	})
	require.NoError(t, err)

	assert.Zero(t, new(big.Int).Sub(controlEscrow, escrowAfter).Cmp(amount),
		"normal ICS20 control must escrow the transferred native amount")
	assert.Zero(t, new(big.Int).Sub(controlSupply.Amount.BigInt(), supplyAfter.Amount.BigInt()).Sign(),
		"normal ICS20 escrow must not change native supply")
	assert.Zero(t, controlContractBalance.Sign(),
		"normal ICS20 control must debit the contract")
	assert.Equal(t, sequenceAfter.NextSequenceSend+1, controlSequence.NextSequenceSend,
		"normal ICS20 control must advance the packet sequence")
	assert.Len(t, controlCommitments.Commitments, 1,
		"normal ICS20 control must leave one packet commitment while the relayer is stopped")
	if len(controlCommitments.Commitments) == 1 {
		assert.Equal(t, controlPacketSequence, controlCommitments.Commitments[0].Sequence)
	}
}

func requireICS20SuccessEvent(
	t *testing.T,
	receipt *ethtypes.Receipt,
	contractAddress common.Address,
	contractABI abi.ABI,
	eventName string,
) uint64 {
	t.Helper()
	event, ok := contractABI.Events[eventName]
	require.True(t, ok, "missing %s ABI", eventName)

	var eventLog *ethtypes.Log
	for _, log := range receipt.Logs {
		if log.Address == contractAddress && len(log.Topics) > 0 && log.Topics[0] == event.ID {
			eventLog = log
			break
		}
	}
	require.NotNil(t, eventLog, "%s must prove the ICS20 call succeeded", eventName)
	require.Len(t, eventLog.Topics, 2)
	require.Equal(t, crypto.Keccak256Hash([]byte("ics20 precompile completed")), eventLog.Topics[1])

	values, err := event.Inputs.NonIndexed().Unpack(eventLog.Data)
	require.NoError(t, err)
	require.Len(t, values, 1)
	packetSequence, ok := values[0].(uint64)
	require.True(t, ok)
	require.NotZero(t, packetSequence)
	return packetSequence
}

func xplaBech32Address(address common.Address) (string, error) {
	fiveBit, err := bech32.ConvertBits(address.Bytes(), 8, 5, true)
	if err != nil {
		return "", err
	}
	return bech32.Encode("xpla", fiveBit)
}
