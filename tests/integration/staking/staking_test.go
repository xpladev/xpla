package staking_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/tests/integration/testutil"
	"github.com/xpladev/xpla/x/staking"
)

func TestDustShare(t *testing.T) {
	input := testutil.CreateTestInput(t)

	sdk.DefaultPowerReduction = sdk.NewInt(1)
	for i := 0; i < 2; i++ {
		err := input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[i].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(100))))
		assert.NoError(t, err)
	}

	// 1 validator & 100 self delegation
	_, err := input.StakingHandler(
		input.Ctx,
		testutil.NewMsgCreateValidator(
			sdk.ValAddress(testutil.Pks[0].Address()),
			testutil.Pks[0],
			sdk.NewInt(100)))
	assert.NoError(t, err)

	staking.EndBlocker(input.Ctx, input.StakingKeeper)
	input.Ctx = input.Ctx.WithBlockHeight(1)

	// slash for dust share
	input.SlashingKeeper.Slash(input.Ctx, sdk.ConsAddress(testutil.Pks[0].Address()), sdk.NewDecWithPrec(1, 2), 100, 1)

	// new 1stake delegator
	_, err = input.StakingHandler(
		input.Ctx,
		testutil.NewMsgDelegate(sdk.AccAddress(testutil.Pks[1].Address()), sdk.ValAddress(testutil.Pks[0].Address()), sdk.NewInt(1)),
	)

	assert.NoError(t, err)

	// try to remove all delegation
	_, err = input.StakingHandler(input.Ctx, stakingtypes.NewMsgUndelegate(sdk.AccAddress(testutil.Pks[0].Address()), sdk.ValAddress(testutil.Pks[0].Address()), sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(99))))
	assert.NoError(t, err)

	delegations := input.StakingKeeper.GetValidatorDelegations(input.Ctx, sdk.ValAddress(testutil.Pks[0].Address()))
	assert.Equal(t, 1, len(delegations))
	assert.Equal(t, sdk.AccAddress(testutil.Pks[1].Address()).String(), delegations[0].DelegatorAddress)
}
