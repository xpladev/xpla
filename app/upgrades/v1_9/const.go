package v1_9

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_9_cube"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Renamed: nil,
		Deleted: []string{},
	},
}
