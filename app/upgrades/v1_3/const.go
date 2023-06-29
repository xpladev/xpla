package v1_3

import (
	icacontrollertypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
)

const (
	UpgradeName = "v1_3"
)

var (
	AddModules = []string{
		icacontrollertypes.StoreKey,
		ibcfeetypes.ModuleName,
	}
)
