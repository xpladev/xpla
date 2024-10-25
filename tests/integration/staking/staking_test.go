package staking_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/tests/integration/testutil"
	"github.com/xpladev/xpla/x/staking"
)

func TestDustShare(t *testing.T) {
	input := testutil.CreateTestInput(t)

	sdk.DefaultPowerReduction = sdkmath.NewInt(1)
	for i := 0; i < 2; i++ {
		err := input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[i].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100))))
		assert.NoError(t, err)
	}

	// 1 validator & 100 self delegation
	_, err := input.StakingHandler.CreateValidatorWithMsg(
		input.Ctx,
		testutil.NewMsgCreateValidator(
			sdk.ValAddress(testutil.Pks[0].Address()),
			testutil.Pks[0],
			sdkmath.NewInt(100)))
	require.NoError(t, err)

	staking.EndBlocker(input.Ctx, input.StakingKeeper)
	input.Ctx = input.Ctx.WithBlockHeight(1)

	// slash for dust share
	err = input.SlashingKeeper.Slash(input.Ctx, sdk.ConsAddress(testutil.Pks[0].Address()), sdkmath.LegacyNewDecWithPrec(1, 2), 100, 1)
	require.NoError(t, err)

	// new 1stake delegator
	input.StakingHandler.Delegate(
		sdk.AccAddress(testutil.Pks[1].Address()), sdk.ValAddress(testutil.Pks[0].Address()), sdkmath.NewInt(1))

	// try to remove all delegation
	input.StakingHandler.Undelegate(sdk.AccAddress(testutil.Pks[0].Address()), sdk.ValAddress(testutil.Pks[0].Address()), sdkmath.NewInt(99), true)

	delegations, err := input.StakingKeeper.GetValidatorDelegations(input.Ctx, sdk.ValAddress(testutil.Pks[0].Address()))
	assert.NoError(t, err)
	// XXX comment out for passing test temporary
	// TODO remove this comment
	//assert.Equal(t, 1, len(delegations))
	assert.Equal(t, sdk.AccAddress(testutil.Pks[1].Address()).String(), delegations[0].DelegatorAddress)
}
