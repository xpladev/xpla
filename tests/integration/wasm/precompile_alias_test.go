package wasm_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	pwasm "github.com/xpladev/xpla/precompile/wasm"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestWasmAliasLifecycleThroughProductionPrecompile(t *testing.T) {
	if runInIsolatedProcess(t) {
		return
	}

	input := testutil.CreateTestInput(t)
	require.NoError(t, input.App.WasmKeeper.SetParams(input.Ctx, wasmtypes.DefaultParams()))

	params := input.App.EvmKeeper.GetParams(input.Ctx)
	params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, pwasm.Address.Hex())
	require.NoError(t, input.App.EvmKeeper.SetParams(input.Ctx, params))

	key, err := crypto.HexToECDSA("7d6bde3ae72b87d4f39b7b731c80c09eb4e9eb31d2b382c48ccf756c3f0e73ad")
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	creator := sdkAddress(sender)
	beneficiary := sdkAddress(common.HexToAddress("0xbEeF00000000000000000000000000000000bEEF"))
	newVerifier := sdkAddress(common.HexToAddress("0x1234000000000000000000000000000000001234"))
	funds, ok := sdkmath.NewIntFromString("20000000000000000000")
	require.True(t, ok)
	require.NoError(t, input.InitAccountWithCoins(creator, sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, funds))))

	wasmCode, err := os.ReadFile(wasmdKeeperTestData(t, "hackatom.wasm"))
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&input.App.WasmKeeper)
	codeID, checksum, err := contractKeeper.Create(input.Ctx, creator, wasmCode, nil)
	require.NoError(t, err)

	receipts := &testutil.EVMReceiptRecorder{}
	input.App.EvmKeeper.SetHooks(receipts)
	deposit := sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(123_456_789)))
	initMsg := mustJSON(t, map[string]string{
		"verifier":    creator.String(),
		"beneficiary": beneficiary.String(),
	})
	supplyBefore := input.BankKeeper.GetSupply(input.Ctx, xplatypes.DefaultDenom)
	result := input.CommitEVMContractCall(
		t, key, receipts, pwasm.Address, pwasm.ABI, "instantiateContract",
		sender, sender, codeID, "production precompile alias", initMsg, cmn.NewCoinsResponse(deposit),
	)
	values, err := pwasm.ABI.Unpack("instantiateContract", result.Response.Ret)
	require.NoError(t, err)
	require.Len(t, values, 2)
	contract20, ok := values[0].(common.Address)
	require.True(t, ok)
	contract32 := requireResolvedPrecompileContract(t, &input, contract20)
	require.NotNil(t, input.App.WasmKeeper.GetContractInfo(input.EVMContext(), contract32))
	require.Equal(t, deposit, input.BankKeeper.GetAllBalances(input.EVMContext(), contract32))
	require.Empty(t, input.BankKeeper.GetAllBalances(input.EVMContext(), sdkAddress(contract20)))
	require.Equal(t, supplyBefore, input.BankKeeper.GetSupply(input.EVMContext(), xplatypes.DefaultDenom))
	requireNativeBankAddressEvent(t, result, banktypes.EventTypeCoinReceived, banktypes.AttributeKeyReceiver, contract32)

	query := input.QueryEVMContract(t, sender, pwasm.Address, pwasm.ABI, "smartContractState", contract20, []byte(`{"verifier":{}}`))
	require.Len(t, query, 1)
	require.JSONEq(t, `{"verifier":"`+creator.String()+`"}`, string(query[0].([]byte)))

	beneficiaryBefore := input.BankKeeper.GetAllBalances(input.EVMContext(), beneficiary)
	result = input.CommitEVMContractCall(
		t, key, receipts, pwasm.Address, pwasm.ABI, "executeContract",
		sender, contract20, []byte(`{"release":{}}`), cmn.NewCoinsResponse(nil),
	)
	require.Equal(t, beneficiaryBefore.Add(deposit...), input.BankKeeper.GetAllBalances(input.EVMContext(), beneficiary))
	require.Empty(t, input.BankKeeper.GetAllBalances(input.EVMContext(), contract32))
	require.Empty(t, input.BankKeeper.GetAllBalances(input.EVMContext(), sdkAddress(contract20)))
	require.Equal(t, supplyBefore, input.BankKeeper.GetSupply(input.EVMContext(), xplatypes.DefaultDenom))
	requireNativeBankAddressEvent(t, result, banktypes.EventTypeCoinSpent, banktypes.AttributeKeySpender, contract32)

	input.CommitEVMContractCall(
		t, key, receipts, pwasm.Address, pwasm.ABI, "migrateContract",
		sender, contract20, codeID, mustJSON(t, map[string]string{"verifier": newVerifier.String()}),
	)
	query = input.QueryEVMContract(t, sender, pwasm.Address, pwasm.ABI, "smartContractState", contract20, []byte(`{"verifier":{}}`))
	require.JSONEq(t, `{"verifier":"`+newVerifier.String()+`"}`, string(query[0].([]byte)))

	t.Run("instantiate2 registers the full address", func(t *testing.T) {
		salt := []byte("precompile-success")
		contract32 := wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, initMsg)
		contract20 := common.BytesToAddress(contract32)
		aliasAddr := sdkAddress(contract20)
		ctx := input.EVMContext()
		aliasFunds := sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(10)))
		require.NoError(t, input.BankKeeper.MintCoins(ctx, minttypes.ModuleName, aliasFunds))
		require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, aliasAddr, aliasFunds))
		supplyBefore := input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom)
		result := input.CommitEVMContractCall(
			t, key, receipts, pwasm.Address, pwasm.ABI, "instantiateContract2",
			sender, sender, codeID, "production precompile instantiate2", initMsg,
			cmn.NewCoinsResponse(deposit), salt, true,
		)
		values, err := pwasm.ABI.Unpack("instantiateContract2", result.Response.Ret)
		require.NoError(t, err)
		returnedAddress, ok := values[0].(common.Address)
		require.True(t, ok)
		require.Equal(t, contract20, returnedAddress)
		require.Equal(t, contract32, requireResolvedPrecompileContract(t, &input, contract20))
		require.NotNil(t, input.App.WasmKeeper.GetContractInfo(input.EVMContext(), contract32))
		require.Equal(t, aliasFunds, input.BankKeeper.GetAllBalances(input.EVMContext(), aliasAddr))
		require.Equal(t, deposit, input.BankKeeper.GetAllBalances(input.EVMContext(), contract32))
		require.Equal(t, supplyBefore, input.BankKeeper.GetSupply(input.EVMContext(), xplatypes.DefaultDenom))
		requireNativeBankAddressEvent(t, result, banktypes.EventTypeCoinReceived, banktypes.AttributeKeyReceiver, contract32)

		beneficiaryBefore := input.BankKeeper.GetAllBalances(input.EVMContext(), beneficiary)
		result = input.CommitEVMContractCall(
			t, key, receipts, pwasm.Address, pwasm.ABI, "executeContract",
			sender, contract20, []byte(`{"release":{}}`), cmn.NewCoinsResponse(nil),
		)
		require.Equal(t, aliasFunds, input.BankKeeper.GetAllBalances(input.EVMContext(), aliasAddr))
		require.Empty(t, input.BankKeeper.GetAllBalances(input.EVMContext(), contract32))
		require.Equal(t, beneficiaryBefore.Add(deposit...), input.BankKeeper.GetAllBalances(input.EVMContext(), beneficiary))
		require.Equal(t, supplyBefore, input.BankKeeper.GetSupply(input.EVMContext(), xplatypes.DefaultDenom))
		requireNativeBankAddressEvent(t, result, banktypes.EventTypeCoinSpent, banktypes.AttributeKeySpender, contract32)
	})

	t.Run("registration conflict rolls back the outer EVM call", func(t *testing.T) {
		salt := []byte("precompile-conflict")
		predicted := wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, initMsg)
		conflictTarget := sdkAddressBytes(append(bytes.Repeat([]byte{0x93}, 12), predicted[len(predicted)-20:]...))
		ctx := input.EVMContext()
		input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, conflictTarget))
		require.NoError(t, input.App.WasmKeeper.RegisterWasmAlias(ctx, predicted[len(predicted)-20:], conflictTarget))

		data, err := pwasm.ABI.Pack(
			"instantiateContract2", sender, sender, codeID, "must rollback", initMsg,
			cmn.NewCoinsResponse(nil), salt, true,
		)
		require.NoError(t, err)
		result := input.CommitEVMTransaction(t, key, receipts, &pwasm.Address, data, testutil.DefaultEVMGasLimit)
		require.NotEmpty(t, result.Response.VmError)
		require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), result.Receipt.Status)

		ctx = input.EVMContext()
		require.Nil(t, input.App.WasmKeeper.GetContractInfo(ctx, predicted))
		require.Nil(t, input.AccountKeeper.GetAccount(ctx, predicted))
		resolved, found, err := input.App.WasmKeeper.ResolveWasmAlias(ctx, predicted[len(predicted)-20:])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, conflictTarget, resolved)
	})

	require.GreaterOrEqual(t, receipts.Len(), 5)
}

func requireNativeBankAddressEvent(t *testing.T, result testutil.EVMTransactionResult, eventType, key string, address sdk.AccAddress) {
	t.Helper()
	for _, event := range result.Events {
		if event.Type != eventType {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == key && attribute.Value == address.String() {
				return
			}
		}
	}
	t.Fatalf("missing %s event with %s=%s", eventType, key, address)
}

func requireResolvedPrecompileContract(t *testing.T, input *testutil.TestInput, address common.Address) sdk.AccAddress {
	t.Helper()
	resolved, found, err := input.App.WasmKeeper.ResolveWasmAlias(input.EVMContext(), address.Bytes())
	require.NoError(t, err)
	require.True(t, found)
	require.Len(t, resolved, 32)
	return resolved
}

func wasmdKeeperTestData(t *testing.T, name string) string {
	t.Helper()
	fn := runtime.FuncForPC(reflect.ValueOf(wasmkeeper.BuildContractAddressClassic).Pointer())
	require.NotNil(t, fn)
	file, _ := fn.FileLine(0)
	require.NotEmpty(t, file)
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	bz, err := json.Marshal(value)
	require.NoError(t, err)
	return bz
}

func sdkAddress(address common.Address) sdk.AccAddress {
	return sdkAddressBytes(address.Bytes())
}

func sdkAddressBytes(address []byte) sdk.AccAddress {
	return sdk.AccAddress(append([]byte(nil), address...))
}
