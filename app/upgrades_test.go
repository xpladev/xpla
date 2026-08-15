package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestRegistersV112Upgrade(t *testing.T) {
	require.Len(t, Upgrades, 1)

	upgrade := Upgrades[0]
	require.Equal(t, "v1_12", upgrade.UpgradeName)
	require.NotNil(t, upgrade.CreateUpgradeHandler)
	require.Equal(t, []string{dynamicdeflationtypes.StoreKey}, upgrade.StoreUpgrades.Added)
	require.Empty(t, upgrade.StoreUpgrades.Renamed)
	require.Empty(t, upgrade.StoreUpgrades.Deleted)
}
