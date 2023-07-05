package volunteer

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type UpgradeVolunteerMsg struct {
	MinGasPrice sdk.Dec `json:"min_gas_price,omitempty"`
}
