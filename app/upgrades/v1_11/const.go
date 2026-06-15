package v1_11

import (
	store "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/xpladev/xpla/app/upgrades"
)

// UpgradeName is the governance upgrade plan name for the v1.11 release.
const UpgradeName = "v1_11"

var StoreUpgrades = store.StoreUpgrades{
	Added:   []string{},
	Renamed: nil,
	Deleted: []string{},
}

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades:        StoreUpgrades,
}
