package v1_13

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const UpgradeName = "v1_13"

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Renamed: nil,
		Deleted: []string{},
	},
}
