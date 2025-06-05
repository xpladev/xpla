package v1_7

import (
	store "cosmossdk.io/store/types"

	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"

	erc20types "github.com/cosmos/evm/x/erc20/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_7"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			ratelimittypes.ModuleName,
		},
		Deleted: []string{
			erc20types.ModuleName,
		},
	},
}
