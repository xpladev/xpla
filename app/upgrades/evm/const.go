package evm

import (
	evmtypes "github.com/xpladev/ethermint/x/evm/types"
	feemarkettypes "github.com/xpladev/ethermint/x/feemarket/types"
)

const (
	UpgradeName = "EVM"
)

var (
	AddModules = []string{evmtypes.ModuleName, feemarkettypes.ModuleName}
)
