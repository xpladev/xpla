package reward_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/abci/types"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	"github.com/xpladev/xpla/x/reward"
	"github.com/xpladev/xpla/x/reward/keeper"
)

// TestBeginBlocker
// 1. 10 validator & 100 self delegation
// 2. validator settlement have 100 & delegation 10, each validator
// 3. 1.1 fee
// 4. process 1 block
func TestBeginBlocker(t *testing.T) {
	input := keeper.CreateTestInput(t)
	sh := staking.NewHandler(input.StakingKeeper)

	sdk.DefaultPowerReduction = sdk.NewIntFromUint64(1000000000000000000)
	defaultFee := sdk.NewInt(11).Mul(sdk.DefaultPowerReduction).Quo(sdk.NewInt(10)) // 1.1

	// create validator & self delegation
	for i := 0; i < keeper.ValidatorCount; i++ {
		err := input.InitAccountWithCoins(sdk.AccAddress(keeper.Pks[i].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100))))
		require.NoError(t, err)

		valAddress := sdk.ValAddress(keeper.Pks[i].Address())
		_, err = sh(input.Ctx, keeper.NewTestMsgCreateValidator(valAddress, keeper.Pks[i], input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100)))
		require.NoError(t, err)
	}

	// validator settlement delegation
	err := input.InitAccountWithCoins(sdk.AccAddress(keeper.Pks[keeper.ValidatorSettlementIndex].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 100))))
	require.NoError(t, err)

	for i := 0; i < keeper.ValidatorCount; i++ {
		valAddress := sdk.ValAddress(keeper.Pks[i].Address())

		_, err = sh(input.Ctx, keeper.NewTestMsgDelegate(sdk.AccAddress(keeper.Pks[keeper.ValidatorSettlementIndex].Address()), valAddress, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 10)))
		require.NoError(t, err)
	}

	staking.EndBlocker(input.Ctx, input.StakingKeeper)

	// checkt balance & staking
	for i := 0; i < keeper.ValidatorCount; i++ {
		require.Equal(
			t, sdk.NewCoins(sdk.Coin{
				Denom:  "",
				Amount: sdk.ZeroInt(),
			}),
			input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(keeper.Pks[i].Address())),
		)

		valAddress := sdk.ValAddress(keeper.Pks[i].Address())
		require.Equal(
			t, input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 110),
			input.StakingKeeper.Validator(input.Ctx, valAddress).GetBondedTokens(),
		)
	}

	require.Equal(
		t, sdk.NewCoins(sdk.Coin{
			Denom:  "",
			Amount: sdk.ZeroInt(),
		}),
		input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(keeper.Pks[keeper.ValidatorSettlementIndex].Address())),
	)

	// fund fee
	err = input.InitAccountWithCoins(sdk.AccAddress(keeper.Pks[keeper.TempIndex].Address()), sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, defaultFee)))
	require.NoError(t, err)

	err = input.BankKeeper.SendCoinsFromAccountToModule(input.Ctx, sdk.AccAddress(keeper.Pks[keeper.TempIndex].Address()), authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, defaultFee)))
	require.NoError(t, err)

	// distirubte
	input.DistrKeeper.SetPreviousProposerConsAddr(input.Ctx, sdk.ConsAddress(keeper.Pks[0].Address()))
	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 2)

	voteInfoes := []types.VoteInfo{}
	for i := 0; i < keeper.ValidatorCount; i++ {
		voteInfoes = append(voteInfoes, types.VoteInfo{
			Validator: types.Validator{
				Address: keeper.Pks[i].Address().Bytes(),
				Power:   int64(110),
			},
		})
	}

	distribution.BeginBlocker(input.Ctx, types.RequestBeginBlock{
		Header: tmproto.Header{
			ProposerAddress: keeper.Pks[0].Address().Bytes(),
		},
		LastCommitInfo: types.LastCommitInfo{
			Round: int32(1),
			Votes: voteInfoes,
		},
	}, input.DistrKeeper)
	reward.BeginBlocker(input.Ctx, types.RequestBeginBlock{}, input.RewardKeeper, input.BankKeeper, input.StakingKeeper, input.DistrKeeper)

	// check result

	// 1. reward module account (0.018)
	decPoolBalance, _ := sdk.NewDecFromStr("0.018")
	poolBalance := decPoolBalance.MulInt(sdk.DefaultPowerReduction)
	blockPerYear := int64(input.RewardKeeper.GetBlocksPerYear(input.Ctx))
	remainPoolBalance := poolBalance.MulInt64(blockPerYear - 1).QuoInt64(blockPerYear).Ceil()
	require.Equal(
		t, remainPoolBalance.TruncateInt(),
		input.RewardKeeper.PoolBalances(input.Ctx)[0].Amount,
	)

	// 2. community pool balance (0.0711)
	res := input.DistrKeeper.GetFeePoolCommunityCoins(input.Ctx)
	communityPool, _ := res.TruncateDecimal()
	require.Equal(
		t, "71100000000000000stake",
		communityPool.String(),
	)

	// 3. reserve account (0.0009)
	require.Equal(
		t, "900000000000000stake",
		input.BankKeeper.GetAllBalances(input.Ctx, sdk.AccAddress(keeper.Pks[keeper.ReserveIndex].Address())).String(),
	)
}
