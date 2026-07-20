package app

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppRegistersV111Upgrade(t *testing.T) {
	upgradeNames := make([]string, 0, len(Upgrades))
	for _, upgrade := range Upgrades {
		upgradeNames = append(upgradeNames, upgrade.UpgradeName)
	}

	require.Contains(t, upgradeNames, "v1_11")
}
