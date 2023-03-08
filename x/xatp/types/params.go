package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const ()

var (
	DefaultTaxRate = sdk.NewDecWithPrec(20, 2) // 20%
)

// Parameter keys
var (
	ParamStoreKeyTaxRate = []byte("taxrate")
)

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default xatp parameters
func DefaultParams() Params {
	return Params{
		TaxRate: DefaultTaxRate,
	}
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyTaxRate, &p.TaxRate, validateTaxRate),
	}
}

func (p Params) Validate() error {
	return nil
}

func validateTaxRate(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid tax rate parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("tax rate must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("tax rate must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("tax rate too large: %s", v)
	}

	return nil
}
