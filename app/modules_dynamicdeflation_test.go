package app

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/app/keepers"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestDynamicDeflationModuleAccountPermissions(t *testing.T) {
	_, hasModuleAccount := maccPerms[dynamicdeflationtypes.ModuleName]
	require.False(t, hasModuleAccount)
	require.Equal(t, []string{authtypes.Burner}, maccPerms[dynamicdeflationtypes.PoolName])
}

func TestDynamicDeflationPoolAccountIsBlocked(t *testing.T) {
	app := &XplaApp{}
	poolAddress := authtypes.NewModuleAddress(dynamicdeflationtypes.PoolName).String()

	require.True(t, app.BlockedModuleAccountAddrs(app.ModuleAccountAddrs())[poolAddress])
}

func TestDynamicDeflationStoreKey(t *testing.T) {
	appKeepers := keepers.AppKeepers{}
	appKeepers.GenerateKeys()

	require.NotNil(t, appKeepers.GetKey(dynamicdeflationtypes.StoreKey))
}

func TestDynamicDeflationBeginBlockOrder(t *testing.T) {
	order := orderBeginBlockers()
	mintIndex := moduleIndex(t, order, minttypes.ModuleName)
	dynamicDeflationIndex := moduleIndex(t, order, dynamicdeflationtypes.ModuleName)
	distributionIndex := moduleIndex(t, order, distrtypes.ModuleName)

	require.Equal(t, mintIndex+1, dynamicDeflationIndex)
	require.Equal(t, dynamicDeflationIndex+1, distributionIndex)
}

func TestDynamicDeflationInitGenesisOrder(t *testing.T) {
	moduleIndex(t, orderInitBlockers(), dynamicdeflationtypes.ModuleName)
}

func moduleIndex(t *testing.T, order []string, moduleName string) int {
	t.Helper()
	for i, name := range order {
		if name == moduleName {
			return i
		}
	}

	require.FailNow(t, "module not found in order", moduleName)
	return -1
}
