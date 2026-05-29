package v1_10

import (
	store "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_10"
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
