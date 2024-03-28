package v1_4

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_4"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{},
	},
}
