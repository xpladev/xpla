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
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

// Exercise a signed native Wasm execution whose BankMsg reaches the real bank
// and EVM keepers, without entering through an EVM precompile.
func TestWasmNativeXerc20Transfer(t *testing.T) {
	if runInIsolatedProcess(t) {
		return
	}
	input := testutil.CreateTestInput(t)
	require.NoError(t, input.App.WasmKeeper.SetParams(input.Ctx, wasmtypes.DefaultParams()))
	key, err := crypto.HexToECDSA("7d6bde3ae72b87d4f39b7b731c80c09eb4e9eb31d2b382c48ccf756c3f0e73ad")
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	creator := sdkAddress(sender)
	require.NoError(t, input.InitAccountWithCoins(creator, sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(20).MulRaw(1_000_000_000_000_000_000)))))

	code, err := os.ReadFile("../../solidity/suites/misc/xerc20_bank_send.wasm")
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&input.App.WasmKeeper)
	codeID, _, err := contractKeeper.Create(input.Ctx, creator, code, nil)
	require.NoError(t, err)
	result := commitCosmosMsg(t, &input, key, &wasmtypes.MsgInstantiateContract{
		Sender: creator.String(), CodeID: codeID, Label: "native xerc20 send", Msg: []byte(`{}`),
	})
	require.Zero(t, result.Code, result.Log)
	contract := requireNewContract(t, result.Events, nil)
	require.Len(t, contract, 32)
	requireRegisteredAlias(t, &input, contract)
	suffix := common.BytesToAddress(contract)
	recipient := common.HexToAddress("0xbEeF00000000000000000000000000000000bEEF")

	raw, err := os.ReadFile("../../solidity/suites/precompiles/artifacts/contracts/test/BankXerc20DoubleSpendPoC.sol/PoCToken.json")
	require.NoError(t, err, "run make contracts-compile from the repository root")
	var artifact testutil.ContractArtifact
	require.NoError(t, json.Unmarshal(raw, &artifact))
	receipts := &testutil.EVMReceiptRecorder{}
	input.App.EvmKeeper.SetHooks(receipts)
	token, tokenABI := input.DeployEVMContract(t, key, receipts, map[string]testutil.ContractArtifact{"PoCToken": artifact}, "PoCToken", big.NewInt(1000))
	input.CommitEVMContractCall(t, key, receipts, token, tokenABI, "transfer", suffix, big.NewInt(100))
	ctx := input.EVMContext()
	wasmAccount := input.AccountKeeper.GetAccount(ctx, contract)
	const wasmSequence = uint64(11)
	require.NoError(t, wasmAccount.SetSequence(wasmSequence))
	input.AccountKeeper.SetAccount(ctx, wasmAccount)

	message := func(to common.Address, amount string) *wasmtypes.MsgExecuteContract {
		return &wasmtypes.MsgExecuteContract{
			Sender: creator.String(), Contract: contract.String(), Msg: mustJSON(t, map[string]any{
				"send_xerc20": map[string]string{"token": strings.ToLower(token.Hex()), "recipient": sdkAddress(to).String(), "amount": amount},
			}),
		}
	}
	balance := func(t *testing.T, address common.Address, want int64) {
		t.Helper()
		values := input.QueryEVMContract(t, sender, token, tokenABI, "balanceOf", address)
		require.Zero(t, big.NewInt(want).Cmp(values[0].(*big.Int)))
	}

	if !t.Run("rejected call does not retain a new suffix account", func(t *testing.T) {
		require.Nil(t, input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix)))
		nativeBefore := input.BankKeeper.GetBalance(input.EVMContext(), contract, xplatypes.DefaultDenom)
		msg := message(common.Address{}, "10")
		msg.Funds = sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 123))
		result := commitCosmosMsg(t, &input, key, msg)
		require.NotZero(t, result.Code)
		require.Contains(t, result.Log, "execution reverted")
		balance(t, suffix, 100)
		balance(t, common.Address{}, 0)
		require.Nil(t, input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix)))
		require.Equal(t, nativeBefore, input.BankKeeper.GetBalance(input.EVMContext(), contract, xplatypes.DefaultDenom))
		require.Equal(t, wasmSequence, input.AccountKeeper.GetAccount(input.EVMContext(), contract).GetSequence())
	}) {
		return
	}

	if !t.Run("absent suffix account", func(t *testing.T) {
		require.Nil(t, input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix)))
		result := commitCosmosMsg(t, &input, key, message(recipient, "10"))
		require.Zero(t, result.Code, result.Log)
		balance(t, suffix, 90)
		balance(t, recipient, 10)
		// Normal EVM execution materializes the sender with nonce zero; it must
		// not borrow the full Wasm account sequence.
		suffixAccount := input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix))
		require.NotNil(t, suffixAccount)
		require.Equal(t, sdkAddress(suffix), suffixAccount.GetAddress())
		require.Zero(t, suffixAccount.GetSequence())
		require.Equal(t, wasmSequence, input.AccountKeeper.GetAccount(input.EVMContext(), contract).GetSequence())
	}) {
		return
	}

	if !t.Run("existing suffix sequence is preserved", func(t *testing.T) {
		ctx := input.EVMContext()
		account := input.AccountKeeper.GetAccount(ctx, sdkAddress(suffix))
		require.NotNil(t, account)
		require.NoError(t, account.SetSequence(7))
		input.AccountKeeper.SetAccount(ctx, account)
		result := commitCosmosMsg(t, &input, key, message(recipient, "10"))
		require.Zero(t, result.Code, result.Log)
		balance(t, suffix, 80)
		balance(t, recipient, 20)
		require.Equal(t, uint64(7), input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix)).GetSequence())
		require.Equal(t, wasmSequence, input.AccountKeeper.GetAccount(input.EVMContext(), contract).GetSequence())
	}) {
		return
	}

	t.Run("ERC20 rejection rolls back attached native funds", func(t *testing.T) {
		nativeBefore := input.BankKeeper.GetBalance(input.EVMContext(), contract, xplatypes.DefaultDenom)
		msg := message(common.Address{}, "10") // Standard ERC20 rejects a zero recipient.
		msg.Funds = sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 123))
		result := commitCosmosMsg(t, &input, key, msg)
		require.NotZero(t, result.Code)
		require.Contains(t, result.Log, "execution reverted")
		balance(t, suffix, 80)
		balance(t, recipient, 20)
		balance(t, common.Address{}, 0)
		require.Equal(t, nativeBefore, input.BankKeeper.GetBalance(input.EVMContext(), contract, xplatypes.DefaultDenom))
		require.Equal(t, uint64(7), input.AccountKeeper.GetAccount(input.EVMContext(), sdkAddress(suffix)).GetSequence())
		require.Equal(t, wasmSequence, input.AccountKeeper.GetAccount(input.EVMContext(), contract).GetSequence())
	})
}
