package app

import (
	"bytes"
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	xplatypes "github.com/xpladev/xpla/types"
)

func TestBankKeeperVirtualAccountsUseMountedObjectStore(t *testing.T) {
	xpla := newTestApp(t)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{})

	sender := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	xpla.AccountKeeper.SetAccount(ctx, xpla.AccountKeeper.NewAccountWithAddress(ctx, sender))
	xpla.AccountKeeper.SetModuleAccount(ctx, authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName))

	startingBalance := sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(1_000))
	virtualFees := sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(300)))
	require.NoError(t, xpla.BankKeeper.UncheckedSetBalance(ctx, sender, startingBalance))

	require.NoError(t, xpla.BankKeeper.SendCoinsFromAccountToModuleVirtual(
		ctx,
		sender,
		authtypes.FeeCollectorName,
		virtualFees,
	))
	require.Equal(t, startingBalance.Sub(virtualFees[0]), xpla.BankKeeper.GetBalance(ctx, sender, xplatypes.DefaultDenom))

	feeCollector := xpla.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	require.NotNil(t, feeCollector)
	require.True(t, xpla.BankKeeper.GetBalance(ctx, feeCollector, xplatypes.DefaultDenom).IsZero())

	require.NoError(t, xpla.BankKeeper.CreditVirtualAccounts(ctx))
	require.Equal(t, virtualFees[0], xpla.BankKeeper.GetBalance(ctx, feeCollector, xplatypes.DefaultDenom))
}
