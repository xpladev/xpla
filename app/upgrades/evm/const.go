package evm

import (
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	feemarkettypes "github.com/evmos/evmos/v14/x/feemarket/types"
)

const (
	UpgradeName = "EVM"
)

var (
	AddModules = []string{evmtypes.ModuleName, feemarkettypes.ModuleName}
)
