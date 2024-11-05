package v1_7

import (
	store "cosmossdk.io/store/types"

	erc20types "github.com/xpladev/ethermint/x/erc20/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_7"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{},
		Deleted: []string{
			erc20types.ModuleName,
		},
	},
}
