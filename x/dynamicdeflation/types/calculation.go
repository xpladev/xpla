package types

import (
	"fmt"
	"math"

	sdkmath "cosmossdk.io/math"
)

// CalculateSettlementEndHeight returns the first global interval boundary at or after startHeight.
func CalculateSettlementEndHeight(startHeight int64, interval uint64) (int64, error) {
	if startHeight < 1 {
		return 0, fmt.Errorf("start height must be positive")
	}
	if interval == 0 || interval > math.MaxInt64 {
		return 0, fmt.Errorf("interval must be between one and %d", int64(math.MaxInt64))
	}
	remainder := uint64(startHeight) % interval
	if remainder == 0 {
		return startHeight, nil
	}
	delta := interval - remainder
	if delta > uint64(math.MaxInt64-startHeight) {
		return 0, fmt.Errorf("settlement end height overflows int64")
	}
	return startHeight + int64(delta), nil
}

// CalculateSettlement computes the deterministic settlement split for a completed period.
func CalculateSettlement(gross, allocated, min, max sdkmath.Int) (burn, community sdkmath.Int, err error) {
	for name, value := range map[string]sdkmath.Int{"gross": gross, "allocated": allocated, "min": min, "max": max} {
		if IsNilOrNegativeInt(value) {
			return sdkmath.Int{}, sdkmath.Int{}, fmt.Errorf("%s must be a non-negative integer", name)
		}
	}
	if !max.GT(min) {
		return sdkmath.Int{}, sdkmath.Int{}, fmt.Errorf("max must be greater than min")
	}
	if allocated.GT(gross) {
		return sdkmath.Int{}, sdkmath.Int{}, fmt.Errorf("allocated cannot exceed gross")
	}

	switch {
	case gross.LTE(min):
		community = sdkmath.ZeroInt()
	case gross.GTE(max):
		community = allocated
	default:
		numerator := allocated.BigInt()
		numerator.Mul(numerator, gross.Sub(min).BigInt())
		numerator.Quo(numerator, max.Sub(min).BigInt())
		community = sdkmath.NewIntFromBigInt(numerator)
	}
	return allocated.Sub(community), community, nil
}
