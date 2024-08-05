package v1_6

import (
	store "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "v1_6"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added:   []string{},
		Deleted: []string{},
	},
}

const (
	// https://github.com/Arachnid/deterministic-deployment-proxy
	thirdwebProxy = "0xf8a58085174876e800830186a08080b853604580600e600039806000f350fe7fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffe03601600081602082378035828234f58015156039578182fd5b8082525050506014600cf31ba02222222222222222222222222222222222222222222222222222222222222222a02222222222222222222222222222222222222222222222222222222222222222"
)
