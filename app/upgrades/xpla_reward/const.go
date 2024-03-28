package xpla_reward

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/xpladev/xpla/app/upgrades"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
)

const (
	UpgradeName = "XplaReward"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			rewardtypes.ModuleName,
		},
	},
}
