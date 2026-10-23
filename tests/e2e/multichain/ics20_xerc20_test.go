package multichain_test

import (
	"context"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	transfer "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	channel "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/tests/e2e"
	"github.com/xpladev/xpla/tests/e2e/multichain"
)

// This suite executes the registered precompile, destination proof, MsgTimeout and acknowledgement.
func TestICS20XERC20TimeoutAndAcknowledgement(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	artifacts := make(map[string]multichain.ContractArtifact)
	for name, source := range map[string]string{
		"PoCToken":          "BankXerc20DoubleSpendPoC.sol",
		"ICS20Xerc20Caller": "ICS20Xerc20Caller.sol",
	} {
		path := "../../solidity/suites/precompiles/artifacts/contracts/test/" + source + "/" + name + ".json"
		data, err := os.ReadFile(path)
		require.NoError(t, err, "read %s; run make test-contracts-compile from the repository root", path)
		var artifact multichain.ContractArtifact
		require.NoError(t, json.Unmarshal(data, &artifact), "decode %s", path)
		artifacts[name] = artifact
	}

	ctx := context.Background()
	ibcSetup := multichain.StartXplaChainAndSimdWithIBC(t, ctx, multichain.LocalImage)
	xplaChain, simdChain := ibcSetup.XplaChain, ibcSetup.SimdChain
	xplaUser, simdUser := ibcSetup.XplaUsers[0], ibcSetup.SimdUsers[0]

	xplaChannels, err := ibcSetup.GetXplaChannels(ctx)
	require.NoError(t, err)
	require.Len(t, xplaChannels, 1)
	xplaChannel := xplaChannels[0]
	require.NoError(t, ibcSetup.StopRelayer(ctx)) // Stop before either packet exists.

	endpoint, err := xplaChain.GetFullNode().GetHostAddress(ctx, "8545/tcp")
	require.NoError(t, err)
	evm, err := ethclient.DialContext(ctx, endpoint)
	require.NoError(t, err)
	defer evm.Close()
	key, err := multichain.ExportEthWalletKey(ctx, xplaChain, xplaUser)
	require.NoError(t, err)
	cosmosWallet := &e2e.WalletInfo{
		PrivKey: &ethsecp256k1.PrivKey{Key: crypto.FromECDSA(key)},
		EncCfg:  *xplaChain.Config().EncodingConfig,
	}
	evmWallet := e2e.EVMWalletInfo{CosmosWalletInfo: cosmosWallet}
	chainID, err := evm.ChainID(ctx)
	require.NoError(t, err)
	auth, err := evmWallet.NewTransactor(ctx, chainID)
	require.NoError(t, err)
	auth.GasLimit = 6_000_000
	tokenAddress, token, err := multichain.DeployEVMContract(ctx, evm, auth, artifacts["PoCToken"], big.NewInt(10))
	require.NoError(t, err)
	denom := "xerc20:" + tokenAddress.Hex()
	callerAddress, caller, err := multichain.DeployEVMContract(ctx, evm, auth, artifacts["ICS20Xerc20Caller"], tokenAddress, denom)
	require.NoError(t, err)
	tx, err := token.Transact(auth, "transfer", callerAddress, big.NewInt(10))
	require.NoError(t, err)
	_, err = multichain.WaitForSuccessfulEVMReceipt(ctx, evm, tx)
	require.NoError(t, err)
	direct := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	escrow := common.BytesToAddress(transfer.GetEscrowAddress("transfer", xplaChannel.ChannelID))
	query := channel.NewQueryClient(xplaChain.GetFullNode().GrpcConn)
	voucher := transfer.NewDenom(denom, transfer.NewHop("transfer", xplaChannel.Counterparty.ChannelID)).IBCDenom()
	accounts := [4]common.Address{callerAddress, direct, escrow, auth.From}
	testCases := []struct {
		name               string
		timeout            bool
		amount             int64
		sent, settled      [3]int64
		destinationVoucher int64
	}{
		{name: "timeout", timeout: true, amount: 5, sent: [3]int64{4, 1, 5}, settled: [3]int64{9, 1, 0}},
		{name: "acknowledgement", amount: 8, sent: [3]int64{0, 2, 8}, settled: [3]int64{0, 2, 8}, destinationVoucher: 8},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header, err := simdChain.GetFullNode().Client.Block(ctx, nil)
			require.NoError(t, err)
			deadline := header.Block.Time.Add(10 * time.Minute)
			if tc.timeout {
				deadline = header.Block.Time.Add(30 * time.Second)
			}
			tx, err := caller.Transact(auth, "send", xplaChannel.ChannelID, simdUser.FormattedAddress(), uint64(deadline.UnixNano()), direct, big.NewInt(1), big.NewInt(tc.amount))
			require.NoError(t, err)
			receipt, err := multichain.WaitForSuccessfulEVMReceipt(ctx, evm, tx)
			require.NoError(t, err)
			startHeight := receipt.BlockNumber.Int64() + 1
			requireXERC20Balances(t, ctx, token, accounts, tc.sent)
			pending, err := query.PacketCommitments(ctx, &channel.QueryPacketCommitmentsRequest{PortId: "transfer", ChannelId: xplaChannel.ChannelID})
			require.NoError(t, err)
			require.Len(t, pending.Commitments, 1)
			sequence := pending.Commitments[0].Sequence
			requireVoucherBalance(t, ctx, simdChain, simdUser.FormattedAddress(), voucher, 0)
			if tc.timeout {
				require.Eventually(t, func() bool {
					block, err := simdChain.GetFullNode().Client.Block(ctx, nil)
					return err == nil && block.Block.Time.After(deadline)
				}, 2*time.Minute, time.Second, "destination committed header must pass timeout")
			}
			require.NoError(t, ibcSetup.FlushRelayer(ctx))
			require.NoError(t, multichain.WaitForBlocks(ctx, xplaChain, 2))
			endHeight, err := xplaChain.Height(ctx)
			require.NoError(t, err)
			destinationQuery := channel.NewQueryClient(simdChain.GetFullNode().GrpcConn)
			var timeoutMsg *channel.MsgTimeout
			var ackMsg *channel.MsgAcknowledgement
			for height := startHeight; height <= endHeight; height++ {
				results, err := xplaChain.GetFullNode().Client.BlockResults(ctx, &height)
				require.NoError(t, err)
				block, err := xplaChain.GetFullNode().Client.Block(ctx, &height)
				require.NoError(t, err)
				for i, result := range results.TxsResults {
					t.Logf("source committed tx=%X height=%d code=%d", block.Block.Txs[i].Hash(), height, result.Code)
					require.Zero(t, result.Code, "source tx failed at height %d: %s", height, result.Log)
				}
				err = cosmos.RangeBlockMessages(ctx, xplaChain.Config().EncodingConfig.InterfaceRegistry, xplaChain.GetFullNode().Client, height, func(msg sdk.Msg) bool {
					switch m := msg.(type) {
					case *channel.MsgTimeout:
						if m.Packet.SourceChannel == xplaChannel.ChannelID && m.Packet.Sequence == sequence {
							timeoutMsg = m
						}
					case *channel.MsgAcknowledgement:
						if m.Packet.SourceChannel == xplaChannel.ChannelID && m.Packet.Sequence == sequence {
							ackMsg = m
						}
					}
					return false
				})
				require.NoError(t, err)
			}
			pending, err = query.PacketCommitments(ctx, &channel.QueryPacketCommitmentsRequest{PortId: "transfer", ChannelId: xplaChannel.ChannelID})
			require.NoError(t, err)
			require.Empty(t, pending.Commitments)
			if tc.timeout {
				require.NotNil(t, timeoutMsg, "real MsgTimeout must be included on source chain")
				require.NotEmpty(t, timeoutMsg.ProofUnreceived)
				proofHeight := int64(timeoutMsg.ProofHeight.RevisionHeight)
				proofBlock, err := simdChain.GetFullNode().Client.Block(ctx, &proofHeight)
				require.NoError(t, err)
				require.True(t, proofBlock.Block.Time.After(deadline), "timeout proof consensus timestamp must exceed deadline")
				received, err := destinationQuery.PacketReceipt(ctx, &channel.QueryPacketReceiptRequest{PortId: "transfer", ChannelId: xplaChannel.Counterparty.ChannelID, Sequence: sequence})
				require.NoError(t, err)
				require.False(t, received.Received)
				require.Nil(t, ackMsg)
				requireXERC20Balances(t, ctx, token, accounts, tc.settled)
				requireVoucherBalance(t, ctx, simdChain, simdUser.FormattedAddress(), voucher, tc.destinationVoucher)
				// Resubmit the exact packet and proof with a funded signer. IBC v10's
				// redundant-relay ante handler must reject it before a second refund.
				timeoutMsg.Signer = xplaUser.FormattedAddress()
				builder := xplaChain.Config().EncodingConfig.TxConfig.NewTxBuilder()
				require.NoError(t, builder.SetMsgs(timeoutMsg))
				builder.SetGasLimit(1_000_000)
				builder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(multichain.Denom, sdkmath.NewInt(280_000_000_000_000_000))))
				conn := xplaChain.GetFullNode().GrpcConn
				cosmosWallet.AccountNumber, cosmosWallet.Sequence, err = e2e.GetAccountNumber(ctx, conn, xplaUser.FormattedAddress())
				require.NoError(t, err)
				txBytes, err := cosmosWallet.SignTx(ctx, xplaChain.Config().ChainID, builder)
				require.NoError(t, err)
				response, err := e2e.BroadcastTx(ctx, conn, txBytes, txtypes.BroadcastMode_BROADCAST_MODE_SYNC)
				require.NoError(t, err)
				require.Equal(t, uint32(22), response.Code)
				require.Contains(t, response.RawLog, channel.ErrRedundantTx.Error())
				t.Logf("timeout replay rejected as redundant tx=%s code=%d", response.TxHash, response.Code)
				requireXERC20Balances(t, ctx, token, accounts, tc.settled)
				requireVoucherBalance(t, ctx, simdChain, simdUser.FormattedAddress(), voucher, tc.destinationVoucher)
			} else {
				require.NotNil(t, ackMsg, "real MsgAcknowledgement must be included on source chain")
				require.Nil(t, timeoutMsg)
				var ack channel.Acknowledgement
				require.NoError(t, channel.SubModuleCdc.UnmarshalJSON(ackMsg.Acknowledgement, &ack))
				require.True(t, ack.Success(), "acknowledgement: %s", ackMsg.Acknowledgement)
				requireXERC20Balances(t, ctx, token, accounts, tc.settled)
				requireVoucherBalance(t, ctx, simdChain, simdUser.FormattedAddress(), voucher, tc.destinationVoucher)
			}
			t.Logf("%s: packet=%s/%d source blocks=%d..%d commitment deleted", tc.name, xplaChannel.ChannelID, sequence, startHeight, endHeight)
		})
	}
}

func requireXERC20Balances(t *testing.T, ctx context.Context, token *bind.BoundContract, accounts [4]common.Address, expected [3]int64) {
	t.Helper()
	var total big.Int
	for i, account := range accounts[:3] {
		balance := queryERC20Balance(t, ctx, token, account)
		require.Zero(t, balance.Cmp(big.NewInt(expected[i])), "account %s", account)
		total.Add(&total, balance)
	}
	require.Zero(t, queryERC20Balance(t, ctx, token, accounts[3]).Sign(), "deployer balance")
	var supply []interface{}
	err := token.Call(&bind.CallOpts{Context: ctx}, &supply, "totalSupply")
	require.NoError(t, err)
	require.Zero(t, supply[0].(*big.Int).Cmp(big.NewInt(10)))
	require.Zero(t, total.Cmp(supply[0].(*big.Int)), "ERC20 balances must conserve totalSupply")
}

func requireVoucherBalance(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, address, denom string, expected int64) {
	t.Helper()
	balance, err := chain.GetBalance(ctx, address, denom)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(expected), balance)
}

func queryERC20Balance(t *testing.T, ctx context.Context, token *bind.BoundContract, owner common.Address) *big.Int {
	t.Helper()
	var result []interface{}
	err := token.Call(&bind.CallOpts{Context: ctx}, &result, "balanceOf", owner)
	require.NoError(t, err)
	return result[0].(*big.Int)
}
