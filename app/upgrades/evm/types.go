package evm

import (
	evmtypes "github.com/xpladev/ethermint/x/evm/types"
	feemarkettypes "github.com/xpladev/ethermint/x/feemarket/types"
)

type EvmUpgradeParams struct {
	Evm       evmtypes.Params       `json:"evm,omitempty"`
	FeeMarket feemarkettypes.Params `json:"fee_market,omitempty"`
}
