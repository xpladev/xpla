package v1_11_test

import (
	"strings"
	"testing"

	"cosmossdk.io/log/v2"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/types"

	xplaapp "github.com/xpladev/xpla/app"
	v1_11 "github.com/xpladev/xpla/app/upgrades/v1_11"
	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestApplyEVMV07StatePatchesOnlyMissingLiveFields(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	params := xpla.EvmKeeper.GetParams(ctx)
	params.EvmDenom = xplatypes.DefaultDenom
	params.ExtendedDenomOptions = nil
	params.HistoryServeWindow = 0
	params.ActiveStaticPrecompiles = removePrecompile(params.ActiveStaticPrecompiles, types.ICS02PrecompileAddress)
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))

	require.NoError(t, v1_11.ApplyEVMV07State(ctx, &xpla.AppKeepers))

	params = xpla.EvmKeeper.GetParams(ctx)
	require.Equal(t, xplatypes.DefaultDenom, params.EvmDenom)
	require.NotNil(t, params.ExtendedDenomOptions)
	require.Equal(t, xplatypes.DefaultDenom, params.ExtendedDenomOptions.ExtendedDenom)
	require.Equal(t, uint64(types.DefaultHistoryServeWindow), params.HistoryServeWindow)
	require.Equal(t, xplaprecompile.DefaultActiveStaticPrecompiles(), params.ActiveStaticPrecompiles)

	requireDefaultPreinstalls(t, xpla, ctx)
}

func TestApplyEVMV07StateRejectsUnexpectedDenom(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	params := xpla.EvmKeeper.GetParams(ctx)
	params.EvmDenom = "uxpla"
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))

	err := v1_11.ApplyEVMV07State(ctx, &xpla.AppKeepers)
	require.ErrorContains(t, err, "unexpected evm denom")
}

func TestApplyEVMV07StateInstallsMissingDefaultPreinstalls(t *testing.T) {
	xpla, ctx := setupUpgradeState(t, false)

	require.NoError(t, v1_11.ApplyEVMV07State(ctx, &xpla.AppKeepers))
	requireDefaultPreinstalls(t, xpla, ctx)
}

func TestApplyEVMV07StateDoesNotResurrectEarlierInactivePrecompiles(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	params := xpla.EvmKeeper.GetParams(ctx)
	params.ActiveStaticPrecompiles = removePrecompile(params.ActiveStaticPrecompiles, types.Bech32PrecompileAddress)
	params.ActiveStaticPrecompiles = removePrecompile(params.ActiveStaticPrecompiles, types.ICS02PrecompileAddress)
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))

	require.NoError(t, v1_11.ApplyEVMV07State(ctx, &xpla.AppKeepers))

	params = xpla.EvmKeeper.GetParams(ctx)
	require.Contains(t, params.ActiveStaticPrecompiles, types.ICS02PrecompileAddress)
	require.NotContains(t, params.ActiveStaticPrecompiles, types.Bech32PrecompileAddress)
}

func TestApplyEVMV07StateRejectsConflictingPreinstall(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	params := xpla.EvmKeeper.GetParams(ctx)
	params.ExtendedDenomOptions = nil
	params.HistoryServeWindow = 0
	params.ActiveStaticPrecompiles = removePrecompile(params.ActiveStaticPrecompiles, types.ICS02PrecompileAddress)
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))

	preinstall := types.DefaultPreinstalls[0]
	address := common.HexToAddress(preinstall.Address)
	badCodeHash := crypto.Keccak256Hash([]byte("different code"))
	xpla.EvmKeeper.SetCodeHash(ctx, address.Bytes(), badCodeHash.Bytes())

	err := v1_11.ApplyEVMV07State(ctx, &xpla.AppKeepers)
	require.ErrorContains(t, err, "different code hash")

	params = xpla.EvmKeeper.GetParams(ctx)
	require.Nil(t, params.ExtendedDenomOptions)
	require.Zero(t, params.HistoryServeWindow)
	require.NotContains(t, params.ActiveStaticPrecompiles, types.ICS02PrecompileAddress)
}

func setupUpgradeState(t *testing.T, installDefaultPreinstalls ...bool) (*xplaapp.XplaApp, sdk.Context) {
	t.Helper()

	xpla := xplaapp.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		true,
		map[int64]bool{},
		t.TempDir(),
		xplaapp.EmptyAppOptions{},
		xplaapp.EmptyWasmOptions,
	)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{Height: 1})

	params := types.DefaultParams()
	params.EvmDenom = xplatypes.DefaultDenom
	params.ExtendedDenomOptions = &types.ExtendedDenomOptions{
		ExtendedDenom: xplatypes.DefaultDenom,
	}
	params.ActiveStaticPrecompiles = xplaprecompile.DefaultActiveStaticPrecompiles()
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))
	if len(installDefaultPreinstalls) == 0 || installDefaultPreinstalls[0] {
		require.NoError(t, xpla.EvmKeeper.AddPreinstalls(ctx, types.DefaultPreinstalls))
	}

	return xpla, ctx
}

func requireDefaultPreinstalls(t *testing.T, xpla *xplaapp.XplaApp, ctx sdk.Context) {
	t.Helper()

	for _, preinstall := range types.DefaultPreinstalls {
		address := common.HexToAddress(preinstall.Address)
		expectedCode := common.FromHex(preinstall.Code)
		expectedCodeHash := crypto.Keccak256Hash(expectedCode)

		require.Equal(t, expectedCodeHash, xpla.EvmKeeper.GetCodeHash(ctx, address))
		require.Equal(t, expectedCode, xpla.EvmKeeper.GetCode(ctx, expectedCodeHash))
		require.NotNil(t, xpla.AccountKeeper.GetAccount(ctx, address.Bytes()))
	}
}

func removePrecompile(precompiles []string, target string) []string {
	filtered := make([]string, 0, len(precompiles))
	for _, precompile := range precompiles {
		if !strings.EqualFold(precompile, target) {
			filtered = append(filtered, precompile)
		}
	}

	return filtered
}
