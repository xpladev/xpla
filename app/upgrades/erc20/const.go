package erc20

import (
	erc20types "github.com/evmos/evmos/v9/x/erc20/types"
)

const (
	UpgradeName = "ERC20"
)

var (
	AddModules = []string{erc20types.ModuleName}
)
