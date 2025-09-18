package util

import (
	"math/big"
)

// Coin for converting sdk.Coin Amount from sdkmath.Int to *big.Int
type Coin struct {
	Denom  string
	Amount *big.Int
}
