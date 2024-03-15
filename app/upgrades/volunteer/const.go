package volunteer

import (
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	erc20types "github.com/xpladev/ethermint/x/erc20/types"
	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"
)

const (
	UpgradeName = "Volunteer"
)

var (
	AddModules = []string{
		icacontrollertypes.StoreKey,
		ibcfeetypes.ModuleName,
		volunteertypes.ModuleName,
		erc20types.ModuleName,
	}
)
