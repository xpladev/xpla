package v1_12

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

// UpgradeName is the on-chain software-upgrade plan name for the v1.12 release.
const UpgradeName = "v1_12"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{dynamicdeflationtypes.StoreKey},
	},
}
