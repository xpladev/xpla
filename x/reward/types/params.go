package types

import (
	"fmt"

	yaml "gopkg.in/yaml.v2"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	DefaultReserveAccount          = ""
	DefaultRewardDistributeAccount = ""
)

var (
	DefaultRateFeePool       = sdk.NewDecWithPrec(20, 2) // 20%
	DefaultRateCommunityPool = sdk.NewDecWithPrec(80, 2) // 80%
	DefaultRateReserve       = sdk.NewDecWithPrec(0, 2)  // 0%
)

// DefaultParams returns default reward parameters
func DefaultParams() Params {
	return Params{
		FeePoolRate:             DefaultRateFeePool,
		CommunityPoolRate:       DefaultRateCommunityPool,
		ReserveRate:             DefaultRateReserve,
		ReserveAccount:          DefaultReserveAccount,
		RewardDistributeAccount: DefaultRewardDistributeAccount,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func (p Params) TotalRate() sdk.Dec {
	return p.CommunityPoolRate.Add(p.FeePoolRate).Add(p.ReserveRate)
}

// ValidateBasic performs basic validation on reward parameters.
func (p Params) ValidateBasic() error {
	if p.ReserveAccount == "" && p.ReserveRate.GT(sdk.ZeroDec()) {
		return fmt.Errorf("reserve account must be set up for reserve compensation")
	}

	if p.CommunityPoolRate.IsNegative() {
		return fmt.Errorf(
			"community pool rate should be positive: %s", p.CommunityPoolRate,
		)
	}

	if p.FeePoolRate.IsNegative() {
		return fmt.Errorf(
			"fee pool rate should be positive: %s", p.FeePoolRate,
		)
	}

	if p.ReserveRate.IsNegative() {
		return fmt.Errorf(
			"reserve rate should be positive: %s", p.ReserveRate,
		)
	}

	if p.TotalRate().GT(sdk.OneDec()) {
		return fmt.Errorf(
			"sum of fee pool, community pool and reserve cannot be greater than one: %s", p.TotalRate(),
		)
	}

	return nil
}

func validateFeePoolRate(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid fee pool rate parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("fee pool rate must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("fee pool rate must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("fee pool rate too large: %s", v)
	}

	return nil
}

func validateCommunityPoolRate(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("invalid community pool rate parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("community pool rate must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("community pool rate must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("community pool rate too large: %s", v)
	}

	return nil
}

func validateReserveRate(i interface{}) error {
	v, ok := i.(sdk.Dec)
	if !ok {
		return fmt.Errorf("reserve rate parameter type: %T", i)
	}

	if v.IsNil() {
		return fmt.Errorf("reserve rate must be not nil")
	}
	if v.IsNegative() {
		return fmt.Errorf("reserve rate must be positive: %s", v)
	}
	if v.GT(sdk.OneDec()) {
		return fmt.Errorf("reserve rate too large: %s", v)
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
