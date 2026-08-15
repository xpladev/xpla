package keeper

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestCalculateAllocatedAmount(t *testing.T) {
	maxIntBig := new(big.Int).Lsh(big.NewInt(1), 256)
	maxIntBig.Sub(maxIntBig, big.NewInt(1))
	maxInt := sdkmath.NewIntFromBigInt(maxIntBig)
	tests := []struct {
		name  string
		gross sdkmath.Int
		rate  sdkmath.LegacyDec
		want  sdkmath.Int
	}{
		{"zero rate", sdkmath.NewInt(100), sdkmath.LegacyZeroDec(), sdkmath.ZeroInt()},
		{"twenty percent", sdkmath.NewInt(100), sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.NewInt(20)},
		{"one hundred percent", sdkmath.NewInt(100), sdkmath.LegacyOneDec(), sdkmath.NewInt(100)},
		{"zero gross", sdkmath.ZeroInt(), sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.ZeroInt()},
		{"truncate one axpla", sdkmath.OneInt(), sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.ZeroInt()},
		{"truncate remainder", sdkmath.NewInt(9), sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.OneInt()},
		{"very large", sdkmath.NewIntFromBigInt(newBigInt("99999999999999999999999999999999999999999999999999")), sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.NewIntFromBigInt(newBigInt("19999999999999999999999999999999999999999999999999"))},
		{"max int", maxInt, sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.NewIntFromBigInt(new(big.Int).Quo(maxIntBig, big.NewInt(5)))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateAllocatedAmount(tt.gross, tt.rate)
			require.NoError(t, err)
			require.True(t, got.Equal(tt.want), "got %s want %s", got, tt.want)
			require.True(t, got.Add(tt.gross.Sub(got)).Equal(tt.gross))
		})
	}
}

func TestCalculateSettlement(t *testing.T) {
	xpla := sdkmath.NewIntWithDecimal(1, 18)
	min := xpla.MulRaw(1000)
	max := xpla.MulRaw(10000)
	allocated := xpla.MulRaw(101)
	tests := []struct {
		name          string
		gross         sdkmath.Int
		wantCommunity sdkmath.Int
	}{
		{"below min", min.SubRaw(1), sdkmath.ZeroInt()},
		{"equal min", min, sdkmath.ZeroInt()},
		{"midpoint", xpla.MulRaw(5500), allocated.QuoRaw(2)},
		{"equal max", max, allocated},
		{"above max", max.AddRaw(1), allocated},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			burn, community, err := CalculateSettlement(tt.gross, allocated, min, max)
			require.NoError(t, err)
			require.True(t, community.Equal(tt.wantCommunity), "got %s want %s", community, tt.wantCommunity)
			require.True(t, burn.Add(community).Equal(allocated))
		})
	}
}

func TestCalculateSettlementRoundingAndLargeInt(t *testing.T) {
	large := sdkmath.NewIntFromBigInt(newBigInt("999999999999999999999999999999999999999999999999999999999999"))
	burn, community, err := CalculateSettlement(large.SubRaw(1), large.SubRaw(3), sdkmath.ZeroInt(), large)
	require.NoError(t, err)
	require.True(t, burn.Add(community).Equal(large.SubRaw(3)))
	want := large.SubRaw(3).BigInt()
	want.Mul(want, large.SubRaw(1).BigInt())
	want.Quo(want, large.BigInt())
	require.True(t, community.Equal(sdkmath.NewIntFromBigInt(want)))
}

func newBigInt(value string) *big.Int {
	n, ok := new(big.Int).SetString(value, 10)
	if !ok {
		panic(value)
	}
	return n
}
