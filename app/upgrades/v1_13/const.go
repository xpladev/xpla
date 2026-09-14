package v1_13

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

// UpgradeName is the on-chain software-upgrade plan name for the v1.13 release.
const UpgradeName = "v1_13"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{dynamicdeflationtypes.StoreKey},
	},
}
