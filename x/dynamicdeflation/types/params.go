package types

import (
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"gopkg.in/yaml.v2"
)

var (
	DefaultAllocationRate = sdkmath.LegacyNewDecWithPrec(20, 2)
	DefaultMinFeeAmount   = sdk.NewCoin(TargetDenom, sdkmath.NewIntWithDecimal(10_000, 18))
	DefaultMaxFeeAmount   = sdk.NewCoin(TargetDenom, sdkmath.NewIntWithDecimal(500_000, 18))
)

const DefaultSettlementIntervalBlocks uint64 = 100_000

func DefaultParams() Params {
	return Params{
		Enabled:                  true,
		AllocationRate:           DefaultAllocationRate,
		SettlementIntervalBlocks: DefaultSettlementIntervalBlocks,
		MinFeeAmount:             DefaultMinFeeAmount,
		MaxFeeAmount:             DefaultMaxFeeAmount,
	}
}

func (p Params) String() string {
	out, _ := yaml.Marshal(p)
	return string(out)
}

func (p Params) Validate() error {
	if p.AllocationRate.IsNil() {
		return fmt.Errorf("allocation rate must not be nil")
	}
	if p.AllocationRate.IsNegative() || p.AllocationRate.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("allocation rate must be between zero and one: %s", p.AllocationRate)
	}
	if p.SettlementIntervalBlocks == 0 {
		return fmt.Errorf("settlement interval blocks must be at least one")
	}
	if p.SettlementIntervalBlocks > math.MaxInt64 {
		return fmt.Errorf("settlement interval blocks must not exceed %d", int64(math.MaxInt64))
	}
	if err := validateThresholdCoin("min fee amount", p.MinFeeAmount); err != nil {
		return err
	}
	if err := validateThresholdCoin("max fee amount", p.MaxFeeAmount); err != nil {
		return err
	}
	if !p.MaxFeeAmount.Amount.GT(p.MinFeeAmount.Amount) {
		return fmt.Errorf("max fee amount must be greater than min fee amount")
	}
	return nil
}

func validateThresholdCoin(name string, coin sdk.Coin) error {
	if coin.Denom != TargetDenom {
		return fmt.Errorf("%s denom must be %s", name, TargetDenom)
	}
	if coin.Amount.IsNil() {
		return fmt.Errorf("%s amount must not be nil", name)
	}
	if coin.Amount.IsNegative() {
		return fmt.Errorf("%s amount must not be negative", name)
	}
	return nil
}
