package volunteer

import (
	store "cosmossdk.io/store/types"

	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"

	erc20types "github.com/xpladev/ethermint/x/erc20/types"
	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "Volunteer"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			icacontrollertypes.StoreKey,
			ibcfeetypes.ModuleName,
			volunteertypes.ModuleName,
			erc20types.ModuleName,
		},
	},
}
