package align_gas_price

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type UpgradeAlignGasPriceMsg struct {
	MinGasPrice sdk.Dec `json:"min_gas_price,omitempty"`
}
