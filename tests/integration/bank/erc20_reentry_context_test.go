package bank_test

import (
	"encoding/binary"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	pbank "github.com/xpladev/xpla/precompile/bank"
	"github.com/xpladev/xpla/tests/integration/testutil"
	bankkeeper "github.com/xpladev/xpla/x/bank/keeper"
	banktypes "github.com/xpladev/xpla/x/bank/types"
)

func TestERC20CallbackSeesUnmergedSDKBranch(t *testing.T) {
	input := testutil.CreateTestInput(t)
	parent := input.EVMContext().WithGasMeter(storetypes.NewGasMeter(10_000_000))
	params := input.App.EvmKeeper.GetParams(parent)
	if !containsString(params.ActiveStaticPrecompiles, pbank.Address.Hex()) {
		params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, pbank.Address.Hex())
		require.NoError(t, input.App.EvmKeeper.SetParams(parent, params))
	}

	witness := common.HexToAddress("0x1111000000000000000000000000000000007777")
	token := common.HexToAddress("0x2222000000000000000000000000000000008888")
	sender := sdk.AccAddress(common.HexToAddress("0x3333").Bytes())
	recipient := sdk.AccAddress(common.HexToAddress("0x4444").Bytes())
	const credit = int64(777)
	calldata, err := pbank.ABI.Pack("balance", witness, "axpla")
	require.NoError(t, err)
	require.LessOrEqual(t, len(calldata), 255)

	// transfer() calls the production BANK.balance precompile, stores the
	// observed amount in slot zero, and returns ABI true.
	code := []byte{0x60, byte(len(calldata)), 0x61, 0, 0, 0x60, 0, 0x39}
	code = append(code, 0x60, 32, 0x60, 0, 0x60, byte(len(calldata)), 0x60, 0, 0x60, 0, 0x73)
	code = append(code, pbank.Address.Bytes()...)
	code = append(code, 0x5a, 0xf1, 0x15, 0x61, 0, 0, 0x57)
	failFixup := len(code) - 3
	code = append(code, 0x60, 0, 0x51, 0x60, 0, 0x55)
	code = append(code, 0x60, 1, 0x60, 0, 0x52, 0x60, 32, 0x60, 0, 0xf3)
	failOffset := len(code)
	code = append(code, 0x5b, 0x60, 0, 0x60, 0, 0xfd)
	dataOffset := len(code)
	binary.BigEndian.PutUint16(code[3:5], uint16(dataOffset))
	binary.BigEndian.PutUint16(code[failFixup:failFixup+2], uint16(failOffset))
	code = append(code, calldata...)

	db := statedb.New(parent, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
	db.SetCode(token, code)
	actionCtx, endAction, err := db.BeginNativeAction()
	require.NoError(t, err)
	defer endAction()
	require.NoError(t, db.FlushToCacheCtx())
	child, _ := actionCtx.CacheContext()
	child = banktypes.WithEVMStateDB(child, db)
	coins := sdk.NewCoins(sdk.NewInt64Coin("axpla", credit))
	require.NoError(t, input.BankKeeper.MintCoins(child, minttypes.ModuleName, coins))
	witnessSDK := sdk.AccAddress(witness.Bytes())
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToAccount(child, minttypes.ModuleName, witnessSDK, coins))

	childBalance := input.BankKeeper.GetBalance(child, witnessSDK, "axpla").Amount.Int64()
	parentBalance := input.BankKeeper.GetBalance(actionCtx, witnessSDK, "axpla").Amount.Int64()
	require.Equal(t, credit, childBalance)
	require.Zero(t, parentBalance)

	keeper := bankkeeper.NewErc20Keeper(input.AccountKeeper, input.App.EvmKeeper)
	require.NoError(t, keeper.ExecuteTransfer(child, token, sender, recipient, big.NewInt(1)))
	observed := db.GetState(token, common.Hash{}).Big().Int64()
	t.Logf(
		"child_sdk_balance=%d parent_sdk_balance=%d erc20_callback_observed=%d",
		childBalance,
		parentBalance,
		observed,
	)
	require.Equal(t, childBalance, observed, "ERC20 callback lost the active SDK branch")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
