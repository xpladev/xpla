package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegistersV111Upgrade(t *testing.T) {
	require.Len(t, Upgrades, 1)

	upgrade := Upgrades[0]
	require.Equal(t, "v1_11", upgrade.UpgradeName)
	require.NotNil(t, upgrade.CreateUpgradeHandler)
	require.Empty(t, upgrade.StoreUpgrades.Added)
	require.Empty(t, upgrade.StoreUpgrades.Renamed)
	require.Empty(t, upgrade.StoreUpgrades.Deleted)
}
