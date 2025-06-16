package v1_8

import (
	store "cosmossdk.io/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_8"
	IbcFeeStoreKey = "feeibc"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{},
		Renamed: nil,
		Deleted: []string{
			IbcFeeStoreKey,
		},
	},
}
