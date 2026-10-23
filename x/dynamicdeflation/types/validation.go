package types

import sdkmath "cosmossdk.io/math"

// IsNilOrNegativeInt reports whether value is nil or less than zero.
func IsNilOrNegativeInt(value sdkmath.Int) bool {
	return value.IsNil() || value.IsNegative()
}
