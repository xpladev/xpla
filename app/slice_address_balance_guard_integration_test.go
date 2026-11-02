package app_test

import (
	"math/big"
	"os"
	"os/exec"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	xplaapp "github.com/xpladev/xpla/app"
	apphelpers "github.com/xpladev/xpla/app/helpers"
	xplatypes "github.com/xpladev/xpla/types"
	xplabankkeeper "github.com/xpladev/xpla/x/bank/keeper"
)

func TestBankSendCannotBypassEVMModuleBalanceGuardWithSliceAlias(t *testing.T) {
	// InitChain configures process-global EVM coin metadata. Isolate this
	// scenario from other app genesis tests, as the Wasm integration suite does.
	const subprocessEnv = "XPLA_SLICE_ADDRESS_BALANCE_GUARD_SUBPROCESS"
	if os.Getenv(subprocessEnv) != "1" {
		command := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$", "-test.timeout=60s", "-test.v")
		command.Env = append(os.Environ(), subprocessEnv+"=1")
		output, err := command.CombinedOutput()
		require.NoError(t, err, string(output))
		t.Log(string(output))
		return
	}

	originalHome := xplaapp.DefaultNodeHome
	xplaapp.DefaultNodeHome = t.TempDir()
	t.Cleanup(func() { xplaapp.DefaultNodeHome = originalHome })

	app := apphelpers.Setup(t, "slice-address-balance-guard")
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	ctx := app.NewUncachedContext(false, types.Header{Height: 2})

	sender := sdk.AccAddress(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	govAddress := app.AccountKeeper.GetModuleAddress(govtypes.ModuleName)
	aliasAddress := append(sdk.AccAddress{0x01}, govAddress...)
	funding := sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 102))

	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, funding))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(
		ctx,
		minttypes.ModuleName,
		sender,
		sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 2)),
	))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(
		ctx,
		minttypes.ModuleName,
		govtypes.ModuleName,
		sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 100)),
	))

	_, err := xplabankkeeper.NewMsgServerImpl(app.BankKeeper).Send(
		ctx,
		banktypes.NewMsgSend(
			sender,
			aliasAddress,
			sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 1)),
		),
	)
	require.NoError(t, err)
	require.NotNil(t, app.AccountKeeper.GetAccount(ctx, aliasAddress))

	resolvedGov := app.AccountKeeper.GetAccount(ctx, govAddress)
	_, isModuleAccount := resolvedGov.(sdk.ModuleAccountI)
	assert.True(t, isModuleAccount, "bank recipient creation must not change the gov account identity")

	govBalanceBefore := app.BankKeeper.GetBalance(ctx, govAddress, xplatypes.DefaultDenom)
	err = app.EvmKeeper.SetBalanceWithLocked(
		ctx,
		common.BytesToAddress(govAddress),
		uint256.NewInt(99),
		big.NewInt(0),
	)
	assert.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
	assert.Equal(t, govBalanceBefore, app.BankKeeper.GetBalance(ctx, govAddress, xplatypes.DefaultDenom))
}
