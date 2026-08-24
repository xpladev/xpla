package v1_12

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const UpgradeName = "v1_12"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Renamed: nil,
		Deleted: []string{},
	},
}
