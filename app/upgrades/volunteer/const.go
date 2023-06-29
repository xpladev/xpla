package volunteer

import (
	icacontrollertypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
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
	}
)
