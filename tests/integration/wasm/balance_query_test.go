package wasm_test

import (
	"encoding/json"
	"math/big"
	"os"
	"strings"
	"testing"

	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	pwasm "github.com/xpladev/xpla/precompile/wasm"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestWasmBalanceThroughProductionPrecompile(t *testing.T) {
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
	require.NoError(t, input.InitAccountWithCoins(creator, sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(20).MulRaw(1_000_000_000_000_000_000)))))
	code, err := os.ReadFile(wasmdKeeperTestData(t, "reflect_2_0.wasm"))
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&input.App.WasmKeeper)
	codeID, _, err := contractKeeper.Create(input.Ctx, creator, code, nil)
	require.NoError(t, err)
	receipts := &testutil.EVMReceiptRecorder{}
	input.App.EvmKeeper.SetHooks(receipts)
	path := "../../solidity/suites/precompiles/artifacts/contracts/test/WasmBalanceReader.sol/WasmBalanceReader.json"
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "run make contracts-compile from the repository root")
	var artifact testutil.ContractArtifact
	require.NoError(t, json.Unmarshal(raw, &artifact))
	reader, readerABI := input.DeployEVMContract(t, key, receipts, map[string]testutil.ContractArtifact{"WasmBalanceReader": artifact}, "WasmBalanceReader")
	result := input.CommitEVMContractCall(t, key, receipts, reader, readerABI, "instantiate", codeID)
	values, err := readerABI.Unpack("instantiate", result.Response.Ret)
	require.NoError(t, err)
	contract20 := values[0].(common.Address)
	contract32 := requireResolvedPrecompileContract(t, &input, contract20)
	require.NotNil(t, input.App.WasmKeeper.GetContractInfo(input.EVMContext(), contract32))

	// Fund the full Wasm address and its distinct suffix account independently.
	ibcDenom := "ibc/" + strings.Repeat("A", 64)
	ctx := input.EVMContext()
	for _, entry := range []struct {
		address sdk.AccAddress
		coins   sdk.Coins
	}{
		{contract32, sdk.NewCoins(sdk.NewInt64Coin("axpla", 123), sdk.NewInt64Coin(ibcDenom, 789))},
		{sdkAddress(contract20), sdk.NewCoins(sdk.NewInt64Coin("axpla", 456))},
		{sdkAddress(reader), sdk.NewCoins(sdk.NewInt64Coin("axpla", 1000))},
	} {
		require.NoError(t, input.BankKeeper.MintCoins(ctx, minttypes.ModuleName, entry.coins))
		require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, entry.address, entry.coins))
	}
	read := func(denom string, want int64) {
		t.Helper()
		got := input.QueryEVMContract(t, sender, reader, readerABI, "read", contract20, denom)
		require.Zero(t, big.NewInt(want).Cmp(got[0].(*big.Int)))
		require.Equal(t, big.NewInt(456), got[1], "EVM suffix balance stays distinct")
	}
	read("axpla", 123)
	read(ibcDenom, 789)
	read("unheld", 0)

	t.Run("committed STATICCALL is read only", func(t *testing.T) {
		ctx := input.EVMContext()
		addresses := []sdk.AccAddress{contract32, sdkAddress(contract20), sdkAddress(reader)}
		balances := make([]sdk.Coins, len(addresses))
		sequences := make([]uint64, len(addresses))
		for i, addr := range addresses {
			balances[i] = input.BankKeeper.GetAllBalances(ctx, addr)
			sequences[i] = input.AccountKeeper.GetAccount(ctx, addr).GetSequence()
		}
		supply := input.BankKeeper.GetSupply(ctx, "axpla")
		result := input.CommitEVMContractCall(t, key, receipts, reader, readerABI, "read", contract20, "axpla")
		got, err := readerABI.Unpack("read", result.Response.Ret)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(123), got[0])
		require.Equal(t, big.NewInt(456), got[1])
		require.Empty(t, result.Receipt.Logs, "query emits no EVM logs")
		for _, event := range result.Events {
			require.NotEqual(t, "wasm", event.Type)
			require.NotEqual(t, "execute", event.Type)
			// Fee events belong to the outer transaction, never the queried accounts.
			for _, attribute := range event.Attributes {
				for _, addr := range addresses {
					require.NotEqual(t, addr.String(), attribute.Value, "query must emit no target account events")
				}
			}
		}
		ctx = input.EVMContext()
		for i, addr := range addresses {
			require.Equal(t, balances[i], input.BankKeeper.GetAllBalances(ctx, addr))
			require.Equal(t, sequences[i], input.AccountKeeper.GetAccount(ctx, addr).GetSequence())
		}
		require.Equal(t, supply, input.BankKeeper.GetSupply(ctx, "axpla"))
		require.Equal(t, contract32, requireResolvedPrecompileContract(t, &input, contract20))
	})

	// A normal owner update retaining the same owner accepts funds without
	// sending them elsewhere, so the expected increase is the full deposit.
	executeMsg := mustJSON(t, map[string]interface{}{"change_owner": map[string]string{"owner": sdkAddress(reader).String()}})

	t.Run("same call observes normal funds transfer", func(t *testing.T) {
		result := input.CommitEVMContractCall(t, key, receipts, reader, readerABI, "fundAndRead", contract20, big.NewInt(25), executeMsg, false)
		got, err := readerABI.Unpack("fundAndRead", result.Response.Ret)
		require.NoError(t, err)
		require.Equal(t, []interface{}{big.NewInt(123), big.NewInt(148), big.NewInt(456)}, got)
		require.Equal(t, "148", input.BankKeeper.GetBalance(input.EVMContext(), contract32, "axpla").Amount.String())
		require.Equal(t, "975", input.BankKeeper.GetBalance(input.EVMContext(), sdkAddress(reader), "axpla").Amount.String())
		read("axpla", 148)
	})

	t.Run("outer revert restores balances before fresh query", func(t *testing.T) {
		data, err := readerABI.Pack("fundAndRead", contract20, big.NewInt(17), executeMsg, true)
		require.NoError(t, err)
		result := input.CommitEVMTransaction(t, key, receipts, &reader, data, testutil.DefaultEVMGasLimit)
		require.NotEmpty(t, result.Response.VmError)
		require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), result.Receipt.Status)
		require.Empty(t, result.Receipt.Logs)
		// Match the deliberate outer revert, proving both reads and the funds
		// execution succeeded before rollback instead of an earlier failure.
		require.Contains(t, string(result.Response.Ret), "requested outer revert")
		require.Equal(t, "148", input.BankKeeper.GetBalance(input.EVMContext(), contract32, "axpla").Amount.String())
		require.Equal(t, "975", input.BankKeeper.GetBalance(input.EVMContext(), sdkAddress(reader), "axpla").Amount.String())
		require.Equal(t, contract32, requireResolvedPrecompileContract(t, &input, contract20))
		read("axpla", 148)
		read(ibcDenom, 789)
	})
}
