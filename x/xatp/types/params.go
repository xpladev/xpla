package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const ()

// Parameter keys
var ()

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default xatp parameters
func DefaultParams() Params {
	return Params{}
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{}
}

func (p Params) Validate() error {
	return nil
}
