package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/reward/types"
)

func TestMsgFundFeeCollector(t *testing.T) {
	tests := []struct {
		amount        sdk.Coins
		depositorAddr sdk.AccAddress
		expectPass    bool
	}{
		{sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt())), sdk.AccAddress(make([]byte, 20)), true},
		{sdk.NewCoins(sdk.NewCoin("stake", sdk.OneInt())), sdk.AccAddress{}, false},
		{sdk.Coins{sdk.Coin{Denom: "1", Amount: sdk.OneInt()}}, sdk.AccAddress(make([]byte, 20)), false},
	}

	for i, tc := range tests {
		msg := types.NewMsgFundFeeCollector(tc.amount, tc.depositorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}
