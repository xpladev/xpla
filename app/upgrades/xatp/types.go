package xatp

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	xatptypes "github.com/xpladev/xpla/x/xatp/types"
)

type UpgradeXatpMsg struct {
	XATP        xatptypes.Params `json:"xatp,omitempty"`
	MinGasPrice sdk.Dec          `json:"min_gas_price,omitempty"`
}
