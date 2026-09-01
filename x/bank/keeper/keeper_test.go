package keeper

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestKeeperViewMethodsReturnZeroForMalformedReservedDenom(t *testing.T) {
	keeper := Keeper{}

	tests := []struct {
		name  string
		query func(string) sdk.Coin
	}{
		{
			name: "balance",
			query: func(denom string) sdk.Coin {
				return keeper.GetBalance(context.Background(), sdk.AccAddress{1}, denom)
			},
		},
		{
			name: "supply",
			query: func(denom string) sdk.Coin {
				return keeper.GetSupply(context.Background(), denom)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, denom := range []string{"xerc20:invalid", "xcw20:invalid"} {
				t.Run(denom, func(t *testing.T) {
					var coin sdk.Coin
					require.NotPanics(t, func() {
						coin = tt.query(denom)
					})

					require.True(t, coin.Amount.IsZero())
				})
			}
		})
	}
}
