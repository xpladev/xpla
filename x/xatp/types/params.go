package types

import (
	"fmt"

	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultXATPPayer = ""
)

// Parameter keys
var (
	ParamStoreKeyXATPs     = []byte("xatps")
	ParamStoreKeyXATPPayer = []byte("xatppayer")
)

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{
		XatpPayer: DefaultXATPPayer,
		Xatps:     []XATP{},
	})
}

// DefaultParams returns default reward parameters
func DefaultParams() Params {
	return Params{
		XatpPayer: DefaultXATPPayer,
		Xatps:     []XATP{},
	}
}

/*func (p Params) String() string {
	var str string
	for i := 0; i < len(p.Xatps); i++ {

		str += p.Xatps[i].Denom + "\n" + p.Xatps[i].Contract + "\n" + p.Xatps[i].Pair
	}
	return str
}*/

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyXATPPayer, &p.XatpPayer, validateXplaPayer),
		paramtypes.NewParamSetPair(ParamStoreKeyXATPs, &p.Xatps, validateXATPs),
	}
}

func (p Params) ValidateBasic() error { return nil }

func validateXATPs(i interface{}) error {
	_, ok := i.([]XATP)
	if !ok {
		return fmt.Errorf("invalid cw20 fee contract parameter type: %T", i)
	}

	return nil
}

func validateXplaPayer(i interface{}) error {
	_, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid xpla player parameter type: %T", i)
	}

	return nil
}
