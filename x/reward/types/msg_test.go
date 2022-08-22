package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/reward/types"
)

func TestMsgFundRewardPool(t *testing.T) {
	tests := []struct {
		amount        sdk.Coin
		depositorAddr sdk.AccAddress
		expectPass    bool
	}{
		{sdk.NewCoin("stake", sdk.OneInt()), sdk.AccAddress(make([]byte, 20)), true},
		{sdk.NewCoin("stake", sdk.OneInt()), sdk.AccAddress{}, false},
		{sdk.Coin{Denom: "1", Amount: sdk.OneInt()}, sdk.AccAddress(make([]byte, 20)), false},
	}

	for i, tc := range tests {
		msg := types.NewMsgFundRewardPool(tc.amount, tc.depositorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}
