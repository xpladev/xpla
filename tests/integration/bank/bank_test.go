package bank_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/tests/integration/testutil"
)

func TestCosmosBank(t *testing.T) {
	input := testutil.CreateTestInput(t)

	for i := 0; i < 2; i++ {
		err := input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[i].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))))
		assert.NoError(t, err)
	}

	// cosmos balance
	balance0 := input.BankKeeper.GetBalance(input.Ctx, sdk.AccAddress(testutil.Pks[0].Address()), sdk.DefaultBondDenom)
	assert.Equal(t, sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)), balance0)

	balance1 := input.BankKeeper.GetBalance(input.Ctx, sdk.AccAddress(testutil.Pks[1].Address()), sdk.DefaultBondDenom)
	assert.Equal(t, sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)), balance1)

	// send 1
	err := input.BankKeeper.SendCoins(input.Ctx, sdk.AccAddress(testutil.Pks[0].Address()), sdk.AccAddress(testutil.Pks[1].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(1))))
	assert.NoError(t, err)

	balance1 = input.BankKeeper.GetBalance(input.Ctx, sdk.AccAddress(testutil.Pks[1].Address()), sdk.DefaultBondDenom)
	assert.Equal(t, sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(101)), balance1)
}
