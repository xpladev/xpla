package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultPayer = ""
)

// Parameter keys
var (
	ParamStoreKeyPayer = []byte("payer")
)

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default xatp parameters
func DefaultParams() Params {
	return Params{
		Payer: DefaultPayer,
	}
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyPayer, &p.Payer, validatePayer),
	}
}

func (p Params) Validate() error {
	if err := validatePayer(p.Payer); err != nil {
		return err
	}

	return nil
}

func validatePayer(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid payer parameter type: %T", i)
	}

	if v != "" {
		_, err := sdk.AccAddressFromBech32(v)
		if err != nil {
			return fmt.Errorf("invalid payer account: %s", err.Error())
		}
	}

	return nil
}
