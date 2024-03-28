package v1_5

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_5"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			consensustypes.ModuleName,
			crisistypes.ModuleName,
		},
	},
}
