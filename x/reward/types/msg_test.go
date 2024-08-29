package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/reward/types"
)

func TestMsgFundRewardPool(t *testing.T) {
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
		msg := types.NewMsgFundRewardPool(tc.amount, tc.depositorAddr)
		if tc.expectPass {
			require.Nil(t, msg.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, msg.ValidateBasic(), "test index: %v", i)
		}
	}
}

func TestMsgUpdateValidateBasic(t *testing.T) {
	testCases := []struct {
		name      string
		msgUpdate *types.MsgUpdateParams
		expPass   bool
	}{
		{
			"fail - invalid authority address",
			&types.MsgUpdateParams{
				Authority: "invalid",
				Params:    types.DefaultParams(),
			},
			false,
		},
		{
			"pass - valid msg",
			&types.MsgUpdateParams{
				Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
				Params:    types.DefaultParams(),
			},
			true,
		},
	}

	for i, tc := range testCases {
		if tc.expPass {
			require.Nil(t, tc.msgUpdate.ValidateBasic(), "test index: %v", i)
		} else {
			require.NotNil(t, tc.msgUpdate.ValidateBasic(), "test index: %v", i)
		}

	}
}
