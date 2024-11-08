package reward_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"

	"github.com/cometbft/cometbft/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/xpladev/xpla/tests/integration/testutil"
	"github.com/xpladev/xpla/x/reward"
)

// TestBeginBlocker
// 1. 10 validator & 100 self delegation
// 2. validator settlement have 100 & delegation 10, each validator
// 3. 1.1 fee
// 4. process 1 block
func TestBeginBlocker(t *testing.T) {
	input := testutil.CreateTestInput(t)
	input.StakingHandler.Commission = stakingtypes.NewCommissionRates(sdkmath.LegacyNewDecWithPrec(10, 2), sdkmath.LegacyOneDec(), sdkmath.LegacyOneDec())

	sdk.DefaultPowerReduction = sdkmath.NewIntFromUint64(1000000000000000000)
	defaultFee := sdkmath.NewInt(11).Mul(sdk.DefaultPowerReduction).Quo(sdkmath.NewInt(10)) // 1.1

	// create validator & self delegation
	for i := 0; i < testutil.ValidatorCount; i++ {
		err := input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[i].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100))))
		require.NoError(t, err)

		valAddress := sdk.ValAddress(testutil.Pks[i].Address())
		input.StakingHandler.CreateValidator(valAddress, testutil.Pks[i], input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100), true)
	}

	// validator settlement delegation
	err := input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[testutil.ValidatorSettlementIndex].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100))))
	require.NoError(t, err)

	for i := 0; i < testutil.ValidatorCount; i++ {
		valAddress := sdk.ValAddress(testutil.Pks[i].Address())

		input.StakingHandler.Delegate(sdk.AccAddress(testutil.Pks[testutil.ValidatorSettlementIndex].Address()), valAddress, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 10))
	}

	input.StakingKeeper.Keeper.EndBlocker(input.Ctx)

	// check balance & staking
	for i := 0; i < testutil.ValidatorCount; i++ {
		require.Equal(
			t, sdk.NewCoins(sdk.Coin{
				Denom:  "",
				Amount: sdkmath.ZeroInt(),
			}),
			input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(testutil.Pks[i].Address())),
		)

		valAddress := sdk.ValAddress(testutil.Pks[i].Address())
		val, err := input.StakingKeeper.Validator(input.Ctx, valAddress)
		require.NoError(t, err)
		require.Equal(
			t, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 110),
			val.GetBondedTokens(),
		)
	}

	require.Equal(
		t, sdk.NewCoins(sdk.Coin{
			Denom:  "",
			Amount: sdkmath.ZeroInt(),
		}),
		input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(testutil.Pks[testutil.ValidatorSettlementIndex].Address())),
	)

	// fund fee
	err = input.InitAccountWithCoins(sdk.AccAddress(testutil.Pks[testutil.TempIndex].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, defaultFee)))
	require.NoError(t, err)

	err = input.BankKeeper.SendCoinsFromAccountToModule(input.Ctx, sdk.AccAddress(testutil.Pks[testutil.TempIndex].Address()), authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, defaultFee)))
	require.NoError(t, err)

	// distribute
	input.DistrKeeper.SetPreviousProposerConsAddr(input.Ctx, sdk.ConsAddress(testutil.Pks[0].Address()))
	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 2)

	voteInfos := []types.VoteInfo{}
	for i := 0; i < testutil.ValidatorCount; i++ {
		voteInfos = append(voteInfos, types.VoteInfo{
			Validator: types.Validator{
				Address: testutil.Pks[i].Address().Bytes(),
				Power:   int64(110),
			},
		})
	}
	input.Ctx = input.Ctx.WithVoteInfos(voteInfos)

	distribution.BeginBlocker(input.Ctx, input.DistrKeeper)
	reward.BeginBlocker(input.Ctx, input.RewardKeeper, input.BankKeeper, input.StakingKeeper, input.DistrKeeper)

	// check result

	// 1. reward module account (0.018)
	decPoolBalance, _ := sdkmath.LegacyNewDecFromStr("0.018")
	poolBalance := decPoolBalance.MulInt(sdk.DefaultPowerReduction)
	blockPerYear := int64(input.RewardKeeper.GetBlocksPerYear(input.Ctx))
	remainPoolBalance := poolBalance.MulInt64(blockPerYear - 1).QuoInt64(blockPerYear).Ceil()
	require.Equal(
		t, remainPoolBalance.TruncateInt(),
		input.RewardKeeper.PoolBalances(input.Ctx)[0].Amount,
	)

	// 2. community pool balance (0.0711)
	res, err := input.DistrKeeper.FeePool.Get(input.Ctx)
	communityPool, _ := res.CommunityPool.TruncateDecimal()
	require.Equal(
		t, "71100000000000000stake",
		communityPool.String(),
	)

	// 3. reserve account (0.0009)
	require.Equal(
		t, "900000000000000stake",
		input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(testutil.Pks[testutil.ReserveIndex].Address())).String(),
	)
}
