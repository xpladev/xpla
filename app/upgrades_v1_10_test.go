package app

import (
	"testing"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	v1_10 "github.com/xpladev/xpla/app/upgrades/v1_10"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestV110ApplyEVMDefaultsSkipsMatchingExistingPreinstall(t *testing.T) {
	xpla := newUpgradeTestApp(t)
	ctx := newUpgradeTestContext(t, xpla)
	require.NoError(t, xpla.EvmKeeper.AddPreinstalls(ctx, evmtypes.DefaultPreinstalls[:1]))

	require.NoError(t, v1_10.ApplyEVMDefaults(ctx, &xpla.AppKeepers))

	for _, preinstall := range evmtypes.DefaultPreinstalls {
		address := common.HexToAddress(preinstall.Address)
		expectedCode := common.FromHex(preinstall.Code)
		expectedCodeHash := crypto.Keccak256Hash(expectedCode)
		actualCodeHash := xpla.EvmKeeper.GetCodeHash(ctx, address)

		require.Equal(t, expectedCodeHash, actualCodeHash)
		require.Equal(t, expectedCode, xpla.EvmKeeper.GetCode(ctx, actualCodeHash))
		require.NotNil(t, xpla.AccountKeeper.GetAccount(ctx, sdk.AccAddress(address.Bytes())))
	}
}

func TestV110ApplyEVMDefaultsRejectsAccountCollision(t *testing.T) {
	xpla := newUpgradeTestApp(t)
	ctx := newUpgradeTestContext(t, xpla)
	preinstall := evmtypes.DefaultPreinstalls[0]
	address := common.HexToAddress(preinstall.Address)
	accountAddress := sdk.AccAddress(address.Bytes())
	xpla.AccountKeeper.SetAccount(ctx, xpla.AccountKeeper.NewAccountWithAddress(ctx, accountAddress))

	err := v1_10.ApplyEVMDefaults(ctx, &xpla.AppKeepers)
	require.ErrorContains(t, err, preinstall.Address)
	require.ErrorContains(t, err, "already has an account")
}

func TestV110ApplyEVMDefaultsRejectsCodeHashCollision(t *testing.T) {
	xpla := newUpgradeTestApp(t)
	ctx := newUpgradeTestContext(t, xpla)
	preinstall := evmtypes.DefaultPreinstalls[0]
	address := common.HexToAddress(preinstall.Address)
	xpla.EvmKeeper.SetCodeHash(ctx, address.Bytes(), crypto.Keccak256Hash([]byte("unexpected")).Bytes())

	err := v1_10.ApplyEVMDefaults(ctx, &xpla.AppKeepers)
	require.ErrorContains(t, err, preinstall.Address)
	require.ErrorContains(t, err, "different code hash")
}

func newUpgradeTestApp(t *testing.T) *XplaApp {
	t.Helper()

	return newTestApp(t)
}

func newUpgradeTestContext(t *testing.T, xpla *XplaApp) sdk.Context {
	t.Helper()

	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{})
	params := evmtypes.DefaultParams()
	params.EvmDenom = xplatypes.DefaultDenom
	params.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{ExtendedDenom: xplatypes.DefaultDenom}
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))

	return ctx
}
