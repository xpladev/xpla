package types

import (
	"cosmossdk.io/collections"

	xplatypes "github.com/xpladev/xpla/types"
)

const (
	ModuleName  = "dynamicdeflation"
	PoolName    = "dynamic_deflation_pool"
	StoreKey    = ModuleName
	TargetDenom = xplatypes.DefaultDenom
)

var (
	ParamsPrefix        = collections.NewPrefix(0)
	CurrentPeriodPrefix = collections.NewPrefix(1)
)
