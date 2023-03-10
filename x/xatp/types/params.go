package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"gopkg.in/yaml.v2"
)

const (
	DefaultReserveAccount = ""
)

var (
	DefaultTaxRate = sdk.NewDecWithPrec(20, 2) // 20%

	DefaultRateFeePool       = sdk.NewDecWithPrec(8333, 4) // 83.33%
	DefaultRateCommunityPool = sdk.NewDecWithPrec(1316, 4) // 13.16%
	DefaultRateRewardPool    = sdk.NewDecWithPrec(333, 4)  // 3.33%
	DefaultRateReserve       = sdk.NewDecWithPrec(0, 4)    // 0%
)

// Parameter keys
var (
	ParamStoreKeyTaxRate = []byte("taxrate")

	ParamStoreKeyFeePoolRate       = []byte("feepoolrate")
	ParamStoreKeyCommunityPoolRate = []byte("communitypoolrate")
	ParamStoreKeyRewardPoolRate    = []byte("rewardpoolrate")
	ParamStoreKeyReserveRate       = []byte("reserverate")
	ParamStoreKeyReserveAccount    = []byte("reserveaccount")
)

// ParamKeyTable - Key declaration for parameters
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// DefaultParams returns default xatp parameters
func DefaultParams() Params {
	return Params{
		TaxRate:           DefaultTaxRate,
		FeePoolRate:       DefaultRateFeePool,
		CommunityPoolRate: DefaultRateCommunityPool,
		ReserveRate:       DefaultRateReserve,
		RewardPoolRate:    DefaultRateRewardPool,
		ReserveAccount:    DefaultReserveAccount,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

// ParamSetPairs returns the parameter set pairs.
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(ParamStoreKeyTaxRate, &p.TaxRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyFeePoolRate, &p.FeePoolRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyCommunityPoolRate, &p.CommunityPoolRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyReserveRate, &p.ReserveRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyRewardPoolRate, &p.RewardPoolRate, validateDec),
		paramtypes.NewParamSetPair(ParamStoreKeyReserveAccount, &p.ReserveAccount, validateAccount),
	}
}

func (p Params) ValidateBasic() error {
	if p.ReserveAccount == "" && p.ReserveRate.GT(sdk.ZeroDec()) {
		return fmt.Errorf("reserve account must be set up for reserve compensation")
	}

	totalRate := p.FeePoolRate.Add(p.CommunityPoolRate).Add(p.ReserveRate).Add(p.RewardPoolRate)
	if totalRate.GT(sdk.OneDec()) {
		return fmt.Errorf(
			"sum of fee pool, community pool, reward pool and reserve cannot be greater than one: %s", totalRate,
		)
	}
	return nil
}

func validateDec(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("too large: %s", v)
	}

	return nil
}

func validateAccount(i interface{}) error {
	v, ok := i.(string)
	if !ok {
		return fmt.Errorf("invalid reserve account parameter type: %T", i)
	}

	if v != "" {
		_, err := sdk.AccAddressFromBech32(v)
		if err != nil {
			return fmt.Errorf("invalid reserve account: %s", err.Error())
		}
	}

	return nil
}
