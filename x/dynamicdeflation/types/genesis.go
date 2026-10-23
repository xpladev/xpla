package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
)

func DefaultGenesisState() *GenesisState {
	return &GenesisState{Params: DefaultParams()}
}

func (g GenesisState) Validate() error {
	if err := g.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}
	if g.CurrentPeriod != nil {
		if err := ValidateCurrentPeriod(*g.CurrentPeriod); err != nil {
			return err
		}
	}
	return nil
}

func ValidateCurrentPeriod(period CurrentPeriod) error {
	if err := period.ActiveConfig.Validate(); err != nil {
		return fmt.Errorf("invalid active config: %w", err)
	}
	if !period.ActiveConfig.Enabled {
		return fmt.Errorf("active period config must be enabled")
	}
	if period.StartHeight < 2 || period.EndHeight < period.StartHeight {
		return fmt.Errorf("invalid period height range %d-%d", period.StartHeight, period.EndHeight)
	}
	wantEnd, err := CalculateSettlementEndHeight(period.StartHeight, period.ActiveConfig.SettlementIntervalBlocks)
	if err != nil {
		return err
	}
	if period.EndHeight != wantEnd {
		return fmt.Errorf("period end height must be %d", wantEnd)
	}
	if err := validateNonNegativeInt("gross amount", period.GrossAmount); err != nil {
		return err
	}
	if err := validateNonNegativeInt("allocated amount", period.AllocatedAmount); err != nil {
		return err
	}
	if period.AllocatedAmount.GT(period.GrossAmount) {
		return fmt.Errorf("allocated amount cannot exceed gross amount")
	}
	return nil
}

func validateNonNegativeInt(name string, amount sdkmath.Int) error {
	if amount.IsNil() {
		return fmt.Errorf("%s must not be nil", name)
	}
	if amount.IsNegative() {
		return fmt.Errorf("%s must not be negative", name)
	}
	return nil
}
