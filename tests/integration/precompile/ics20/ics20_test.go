package ics20_test

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	ics20 "github.com/cosmos/evm/precompiles/ics20"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestICS20XERC20CommittedConservation(t *testing.T) {
	input := testutil.CreateTestInput(t)
	app := input.App
	key, err := crypto.GenerateKey()
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	address := sdk.AccAddress(sender.Bytes())
	balance := sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(20).MulRaw(1_000_000_000_000_000_000))
	require.NoError(t, input.InitAccountWithCoins(address, sdk.NewCoins(balance)))

	receipts := &testutil.EVMReceiptRecorder{}
	app.EvmKeeper.SetHooks(receipts)
	artifacts := make(map[string]testutil.ContractArtifact)
	for name, source := range map[string]string{
		"PoCToken":            "BankXerc20DoubleSpendPoC.sol",
		"AdversarialXerc20":   "BankXerc20DoubleSpendPoC.sol",
		"Xerc20ReentryTarget": "BankXerc20DoubleSpendPoC.sol",
		"ICS20Xerc20Caller":   "ICS20Xerc20Caller.sol",
	} {
		path := "../../../solidity/suites/precompiles/artifacts/contracts/test/" + source + "/" + name + ".json"
		data, err := os.ReadFile(path)
		require.NoError(t, err, "read %s; run make test-contracts-compile from the repository root", path)
		var artifact testutil.ContractArtifact
		require.NoError(t, json.Unmarshal(data, &artifact), "decode %s", path)
		artifacts[name] = artifact
	}

	recipient := common.HexToAddress("0x123456")

	token, tokenABI, caller, callerABI, denom := deployICS20Contracts(t, &input, key, receipts, artifacts, "PoCToken")
	result := input.CommitEVMContractCall(t, key, receipts, caller, callerABI, "send", makeICS20SendArgs(recipient, 1, 999)...)
	requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{0, 1, 999})
	values, err := callerABI.Unpack("send", result.Response.Ret)
	require.NoError(t, err)
	require.Equal(t, uint64(1), values[0])
	require.Zero(t, values[1].(*big.Int).Sign(), "same-call balance must see escrow")
	require.False(t, values[2].(bool), "already escrowed tokens cannot be reused")
	require.NotEmpty(t, values[3].([]byte))
	require.NotEmpty(t, app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", 1))
	require.Equal(t, "999", app.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).Amount.String())
	logs := testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["Transfer"].ID)
	require.Len(t, logs, 2, "only direct transfer and successful escrow belong in the receipt")
	escrow := common.BytesToAddress(transfertypes.GetEscrowAddress("transfer", "channel-0"))
	require.Equal(t, common.BytesToHash(caller.Bytes()), logs[1].Topics[1])
	require.Equal(t, common.BytesToHash(escrow.Bytes()), logs[1].Topics[2])
	amount, err := tokenABI.Events["Transfer"].Inputs.NonIndexed().Unpack(logs[1].Data)
	require.NoError(t, err)
	require.Equal(t, big.NewInt(999), amount[0])
	packets := 0
	for _, event := range result.Events {
		if event.Type == "send_packet" {
			packets++
		}
	}
	require.Equal(t, 1, packets)
	t.Run("repeated calls", func(t *testing.T) {
		token, tokenABI, caller, callerABI, denom = deployICS20Contracts(t, &input, key, receipts, artifacts, "PoCToken")
		start := nextICS20Sequence(t, &input)
		args := append(makeICS20SendArgs(recipient, 1, 400), big.NewInt(599))
		result := input.CommitEVMContractCall(t, key, receipts, caller, callerABI, "sendRepeated", args...)
		values, err := callerABI.Unpack("sendRepeated", result.Response.Ret)
		require.NoError(t, err)
		require.Equal(t, []interface{}{start, start + 1}, values)
		requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{0, 1, 999})
		seq, found := app.IBCKeeper.ChannelKeeper.GetNextSequenceSend(input.EVMContext(), "transfer", "channel-0")
		require.True(t, found)
		require.Equal(t, start+2, seq)
		for _, n := range []uint64{start, start + 1} {
			require.NotEmpty(t, app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", n))
		}
		require.Equal(t, "999", app.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).Amount.String())
		logs := testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["Transfer"].ID)
		require.Len(t, logs, 3)
		escrow := common.BytesToAddress(transfertypes.GetEscrowAddress("transfer", "channel-0"))
		for i, amount := range []int64{400, 599} {
			require.Equal(t, common.BytesToHash(caller.Bytes()), logs[i+1].Topics[1])
			require.Equal(t, common.BytesToHash(escrow.Bytes()), logs[i+1].Topics[2])
			values, err := tokenABI.Events["Transfer"].Inputs.NonIndexed().Unpack(logs[i+1].Data)
			require.NoError(t, err)
			require.Equal(t, big.NewInt(amount), values[0])
		}
		count := 0
		for _, event := range result.Events {
			if event.Type == "send_packet" {
				count++
			}
		}
		require.Equal(t, 2, count)
	})
	t.Run("callbacks", func(t *testing.T) {
		for _, tc := range []struct {
			name                                       string
			revertCallback, catchCallback, revertOuter bool
		}{
			{name: "successful callback"},
			{name: "caught callback failure", revertCallback: true, catchCallback: true},
			{name: "uncaught callback failure", revertCallback: true},
			{name: "successful callback outer rollback", revertOuter: true},
		} {
			t.Run(tc.name, func(t *testing.T) {
				token, tokenABI, caller, callerABI, denom = deployICS20Contracts(t, &input, key, receipts, artifacts, "AdversarialXerc20")
				start := nextICS20Sequence(t, &input)
				target, targetABI := input.DeployEVMContract(t, key, receipts, artifacts, "Xerc20ReentryTarget", token, denom)
				input.CommitEVMContractCall(t, key, receipts, target, targetABI, "configure", recipient, big.NewInt(0), tc.revertCallback)
				callback, err := targetABI.Pack("onXerc20Transfer")
				require.NoError(t, err)
				input.CommitEVMContractCall(t, key, receipts, token, tokenABI, "configureCallback", target, callback, tc.catchCallback, big.NewInt(0))
				method := "send"
				if tc.revertOuter {
					method = "sendThenRevert"
				}
				data, err := callerABI.Pack(method, makeICS20SendArgs(recipient, 0, 1000)...)
				require.NoError(t, err)
				result := input.CommitEVMTransaction(t, key, receipts, &caller, data, testutil.DefaultEVMGasLimit)
				failure := tc.revertOuter || (tc.revertCallback && !tc.catchCallback)
				seq, found := app.IBCKeeper.ChannelKeeper.GetNextSequenceSend(input.EVMContext(), "transfer", "channel-0")
				require.True(t, found)
				commitment := app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", start)
				if failure {
					require.NotEmpty(t, result.Response.VmError)
					require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), result.Receipt.Status)
					require.Empty(t, result.Receipt.Logs)
					require.Equal(t, start, seq)
					require.Empty(t, commitment)
					require.True(t, app.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).IsZero())
					for _, event := range result.Events {
						require.NotEqual(t, "send_packet", event.Type)
					}
					requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{1000, 0, 0})
					if tc.revertOuter {
						require.Equal(t, callerABI.Errors["ForcedOuterRevert"].ID.Bytes()[:4], result.Response.Ret)
					}
					return
				}
				require.Empty(t, result.Response.VmError)
				require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
				require.Equal(t, start+1, seq)
				require.NotEmpty(t, commitment)
				require.Equal(t, "1000", app.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).Amount.String())
				requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{0, 0, 1000})
				values, err := callerABI.Unpack("send", result.Response.Ret)
				require.NoError(t, err)
				require.Equal(t, start, values[0])
				require.Zero(t, values[1].(*big.Int).Sign())
				require.False(t, values[2].(bool))
				reason, err := abi.UnpackRevert(values[3].([]byte))
				require.NoError(t, err)
				require.Equal(t, "ERC20: transfer amount exceeds balance", reason)
				callbackLogs := testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["CallbackResult"].ID)
				require.Len(t, callbackLogs, 1, "failed token reuse must not retain its callback logs")
				callbackResult, err := tokenABI.Events["CallbackResult"].Inputs.NonIndexed().Unpack(callbackLogs[0].Data)
				require.NoError(t, err)
				require.Equal(t, !tc.revertCallback, callbackResult[0])
				if tc.revertCallback {
					require.Equal(t, targetABI.Errors["AdversarialCallbackRevert"].ID.Bytes()[:4], callbackResult[1])
					require.Empty(t, testutil.EVMLogsForEvent(result.Receipt, target, targetABI.Events["CallbackEntered"].ID), "reverted callback frame logs must disappear")
				} else {
					require.Empty(t, callbackResult[1])
					require.Len(t, testutil.EVMLogsForEvent(result.Receipt, target, targetABI.Events["CallbackEntered"].ID), 1)
				}
				require.Len(t, testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["PoisonLog"].ID), 1)
				require.Len(t, testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["Transfer"].ID), 1)
			})
		}
	})
	t.Run("rollback", func(t *testing.T) {
		for _, child := range []bool{false, true} {
			name := "outer revert"
			if child {
				name = "caught child revert preserves parent"
			}
			t.Run(name, func(t *testing.T) {
				token, tokenABI, caller, callerABI, denom = deployICS20Contracts(t, &input, key, receipts, artifacts, "PoCToken")
				start := nextICS20Sequence(t, &input)
				method, args := "sendThenRevert", makeICS20SendArgs(recipient, 1, 999)
				if child {
					method = "catchChildRevert"
					args = append(args, big.NewInt(7))
				}
				data, err := callerABI.Pack(method, args...)
				require.NoError(t, err)
				result := input.CommitEVMTransaction(t, key, receipts, &caller, data, testutil.DefaultEVMGasLimit)
				requireICS20TransferRolledBack(t, &input, denom, result, start)
				if !child {
					require.NotEmpty(t, result.Response.VmError)
					require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), result.Receipt.Status)
					require.Equal(t, callerABI.Errors["ForcedOuterRevert"].ID.Bytes()[:4], result.Response.Ret)
					require.Empty(t, result.Receipt.Logs)
					requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{1000, 0, 0})
					return
				}
				require.Empty(t, result.Response.VmError)
				require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
				requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{993, 7, 0})
				require.Len(t, result.Receipt.Logs, 2, "only parent ChildResult and Transfer survive")
				require.Empty(t, testutil.EVMLogsForEvent(result.Receipt, caller, callerABI.Events["SendResult"].ID))
				childLogs := testutil.EVMLogsForEvent(result.Receipt, caller, callerABI.Events["ChildResult"].ID)
				require.Len(t, childLogs, 1)
				values, err := callerABI.Events["ChildResult"].Inputs.Unpack(childLogs[0].Data)
				require.NoError(t, err)
				require.Equal(t, []interface{}{false, callerABI.Errors["ForcedOuterRevert"].ID.Bytes()[:4]}, values)
				logs := testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["Transfer"].ID)
				require.Len(t, logs, 1)
				require.Equal(t, common.BytesToHash(recipient.Bytes()), logs[0].Topics[2])
				amount, err := tokenABI.Events["Transfer"].Inputs.NonIndexed().Unpack(logs[0].Data)
				require.NoError(t, err)
				require.Equal(t, big.NewInt(7), amount[0])
			})
		}
	})
	t.Run("gas", func(t *testing.T) {
		token, tokenABI, caller, callerABI, denom = deployICS20Contracts(t, &input, key, receipts, artifacts, "AdversarialXerc20")
		input.CommitEVMContractCall(t, key, receipts, token, tokenABI, "configureCallback", common.Address{}, []byte{}, false, big.NewInt(1000))
		data, err := callerABI.Pack("send", makeICS20SendArgs(recipient, 0, 1000)...)
		require.NoError(t, err)
		start := nextICS20Sequence(t, &input)
		result := input.CommitEVMTransaction(t, key, receipts, &caller, data, 250_000)
		require.NotEmpty(t, result.Response.VmError)
		reason, err := abi.UnpackRevert(result.Response.Ret)
		require.NoError(t, err)
		require.Contains(t, reason, "out of gas")
		require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), result.Receipt.Status)
		require.Empty(t, result.Receipt.Logs)
		requireICS20TransferRolledBack(t, &input, denom, result, start)
		requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{1000, 0, 0})
		require.Equal(t, [32]byte{}, input.QueryEVMContract(t, sender, token, tokenABI, "burnAccumulator")[0])
		result = input.CommitEVMTransaction(t, key, receipts, &caller, data, testutil.DefaultEVMGasLimit)
		require.Empty(t, result.Response.VmError)
		require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
		requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{0, 0, 1000})
		require.Equal(t, start+1, nextICS20Sequence(t, &input))
		require.NotEmpty(t, app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", start))
		require.Len(t, testutil.EVMLogsForEvent(result.Receipt, token, tokenABI.Events["Transfer"].ID), 1)
	})
	t.Run("native ICS20", func(t *testing.T) {
		address := common.HexToAddress("0x0802")
		data, err := ics20.ABI.Pack(ics20.TransferMethod, "transfer", "channel-0", xplatypes.DefaultDenom,
			big.NewInt(123), sender, makeICS20SendArgs(recipient, 0, 0)[1], clienttypes.ZeroHeight(), uint64(time.Hour.Nanoseconds()), "")
		require.NoError(t, err)
		escrow := transfertypes.GetEscrowAddress("transfer", "channel-0")
		before := app.BankKeeper.GetBalance(input.EVMContext(), escrow, xplatypes.DefaultDenom).Amount
		start := nextICS20Sequence(t, &input)
		result := input.CommitEVMTransaction(t, key, receipts, &address, data, testutil.DefaultEVMGasLimit)
		require.Empty(t, result.Response.VmError)
		require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
		require.Equal(t, before.AddRaw(123), app.BankKeeper.GetBalance(input.EVMContext(), escrow, xplatypes.DefaultDenom).Amount)
		require.Equal(t, start+1, nextICS20Sequence(t, &input))
		require.NotEmpty(t, app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", start))
	})
	t.Run("Cosmos xerc20", func(t *testing.T) {
		// Deploy a separate constructor-funded token owned by the Cosmos signer.
		token, tokenABI = input.DeployEVMContract(t, key, receipts, artifacts, "PoCToken", big.NewInt(1000))
		denom = "xerc20:" + token.Hex()
		caller = sender
		account := app.AccountKeeper.GetAccount(input.EVMContext(), sdk.AccAddress(sender.Bytes()))
		msg := transfertypes.NewMsgTransfer("transfer", "channel-0", sdk.NewCoin(denom, sdkmath.NewInt(1000)),
			account.GetAddress().String(), makeICS20SendArgs(recipient, 0, 0)[1].(string), clienttypes.ZeroHeight(), uint64(time.Hour.Nanoseconds()), "")
		builder := app.GetTxConfig().NewTxBuilder()
		require.NoError(t, builder.SetMsgs(msg))
		builder.SetGasLimit(testutil.DefaultEVMGasLimit)
		builder.SetFeeAmount(sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 10_000_000_000_000_000)))
		priv := &ethsecp256k1.PrivKey{Key: crypto.FromECDSA(key)}
		mode := signing.SignMode_SIGN_MODE_DIRECT
		require.NoError(t, builder.SetSignatures(signing.SignatureV2{PubKey: priv.PubKey(),
			Data: &signing.SingleSignatureData{SignMode: mode}, Sequence: account.GetSequence()}))
		sig, err := clienttx.SignWithPrivKey(input.EVMContext(), mode, authsigning.SignerData{
			ChainID: testutil.TestChainID, AccountNumber: account.GetAccountNumber(), Sequence: account.GetSequence(),
		}, builder, priv, app.GetTxConfig(), account.GetSequence())
		require.NoError(t, err)
		require.NoError(t, builder.SetSignatures(sig))
		txBytes, err := app.GetTxConfig().TxEncoder()(builder.GetTx())
		require.NoError(t, err)
		start := nextICS20Sequence(t, &input)
		input.CommitTransaction(t, txBytes)
		requireICS20Balances(t, &input, sender, token, tokenABI, caller, recipient, [3]int64{0, 0, 1000})
		require.Equal(t, "1000", app.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).Amount.String())
		require.Equal(t, start+1, nextICS20Sequence(t, &input))
		require.NotEmpty(t, app.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", start))
	})
}

func deployICS20Contracts(
	t *testing.T,
	input *testutil.TestInput,
	key *ecdsa.PrivateKey,
	receipts *testutil.EVMReceiptRecorder,
	artifacts map[string]testutil.ContractArtifact,
	tokenName string,
) (common.Address, abi.ABI, common.Address, abi.ABI, string) {
	t.Helper()
	token, tokenABI := input.DeployEVMContract(t, key, receipts, artifacts, tokenName, big.NewInt(1000))
	denom := "xerc20:" + token.Hex()
	caller, callerABI := input.DeployEVMContract(t, key, receipts, artifacts, "ICS20Xerc20Caller", token, denom)
	input.CommitEVMContractCall(t, key, receipts, token, tokenABI, "transfer", caller, big.NewInt(1000))
	return token, tokenABI, caller, callerABI, denom
}

func requireICS20Balances(
	t *testing.T,
	input *testutil.TestInput,
	sender, token common.Address,
	tokenABI abi.ABI,
	caller, recipient common.Address,
	expected [3]int64,
) {
	t.Helper()
	accounts := []common.Address{caller, recipient, common.BytesToAddress(transfertypes.GetEscrowAddress("transfer", "channel-0"))}
	sum := new(big.Int)
	actual := make([]*big.Int, len(accounts))
	for i, account := range accounts {
		actual[i] = input.QueryEVMContract(t, sender, token, tokenABI, "balanceOf", account)[0].(*big.Int)
		sum.Add(sum, actual[i])
	}
	supply := input.QueryEVMContract(t, sender, token, tokenABI, "totalSupply")[0].(*big.Int)
	t.Logf("committed height=%d balances sender=%s recipient=%s escrow=%s sum=%s totalSupply=%s", input.App.LastBlockHeight(), actual[0], actual[1], actual[2], sum, supply)
	require.Equal(t, "1000", supply.String())
	require.Equal(t, supply.String(), sum.String(), "committed ERC20 balances must conserve totalSupply")
	for i := range accounts {
		require.Equal(t, big.NewInt(expected[i]).String(), actual[i].String(), "account %s", accounts[i])
	}
}

func makeICS20SendArgs(recipient common.Address, direct, amount int64) []interface{} {
	return []interface{}{"channel-0", "cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a", uint64(time.Hour.Nanoseconds()), recipient, big.NewInt(direct), big.NewInt(amount)}
}

func nextICS20Sequence(t *testing.T, input *testutil.TestInput) uint64 {
	t.Helper()
	sequence, found := input.App.IBCKeeper.ChannelKeeper.GetNextSequenceSend(input.EVMContext(), "transfer", "channel-0")
	require.True(t, found)
	return sequence
}

func requireICS20TransferRolledBack(
	t *testing.T,
	input *testutil.TestInput,
	denom string,
	result testutil.EVMTransactionResult,
	sequence uint64,
) {
	t.Helper()
	require.Equal(t, sequence, nextICS20Sequence(t, input))
	require.Empty(t, input.App.IBCKeeper.ChannelKeeper.GetPacketCommitment(input.EVMContext(), "transfer", "channel-0", sequence))
	require.True(t, input.App.TransferKeeper.GetTotalEscrowForDenom(input.EVMContext(), denom).IsZero())
	for _, event := range result.Events {
		require.NotEqual(t, "send_packet", event.Type)
	}
}
