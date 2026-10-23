package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

// CalculateAllocatedAmount applies a deterministic decimal multiplication and truncates toward zero.
func CalculateAllocatedAmount(gross sdkmath.Int, rate sdkmath.LegacyDec) (sdkmath.Int, error) {
	if types.IsNilOrNegativeInt(gross) {
		return sdkmath.Int{}, fmt.Errorf("gross amount must be a non-negative integer")
	}
	if rate.IsNil() || rate.IsNegative() || rate.GT(sdkmath.LegacyOneDec()) {
		return sdkmath.Int{}, fmt.Errorf("rate must be between zero and one")
	}
	// LegacyDec stores a fixed-point integer scaled by 10^18. Avoid LegacyDec.MulInt
	// because its checked intermediate can overflow even when the truncated result
	// is bounded by gross.
	numerator := gross.BigInt()
	numerator.Mul(numerator, rate.BigInt())
	numerator.Quo(numerator, sdkmath.LegacyOneDec().BigInt())
	return sdkmath.NewIntFromBigInt(numerator), nil
}

// CalculateSettlement returns burn and community amounts while preserving burn+community=allocated.
func CalculateSettlement(gross, allocated, min, max sdkmath.Int) (burn, community sdkmath.Int, err error) {
	return types.CalculateSettlement(gross, allocated, min, max)
}
