package evm

import (
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	feemarkettypes "github.com/evmos/evmos/v14/x/feemarket/types"
)

type EvmUpgradeParams struct {
	Evm       evmtypes.Params       `json:"evm,omitempty"`
	FeeMarket feemarkettypes.Params `json:"fee_market,omitempty"`
}
