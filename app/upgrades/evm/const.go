package evm

import (
	store "cosmossdk.io/store/types"
	evmtypes "github.com/xpladev/ethermint/x/evm/types"
	feemarkettypes "github.com/xpladev/ethermint/x/feemarket/types"

	"github.com/xpladev/xpla/app/upgrades"
)

const (
	UpgradeName = "EVM"
)

var Upgrade = upgrades.Upgrade{
	UpgradeName:          UpgradeName,
	CreateUpgradeHandler: CreateUpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			evmtypes.ModuleName,
			feemarkettypes.ModuleName,
		},
	},
}
