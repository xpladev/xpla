package reward

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/xpladev/xpla/x/reward/keeper"
	"github.com/xpladev/xpla/x/reward/types"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock, k keeper.Keeper, bk types.BankKeeper, sk types.StakingKeeper, dk types.DistributionKeeper) {
	params := k.GetParams(ctx)

	if params.RewardDistributeAccount == "" {
		return
	}

	total := params.TotalRate()

	rewardDistributeAccount := sdk.MustAccAddressFromBech32(params.RewardDistributeAccount)

	totalRewards := map[string]sdk.Coin{}

	sk.IterateDelegations(ctx, rewardDistributeAccount, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
		validator := delegation.GetValidatorAddr()

		reward, err := dk.WithdrawDelegationRewards(ctx, rewardDistributeAccount, validator)
		if err == disttypes.ErrEmptyDelegationDistInfo {
			return false
		} else if err != nil {
			panic(err)
		}

		for _, coin := range reward {
			if coin.Amount.IsZero() {
				continue
			}

			c, exist := totalRewards[coin.Denom]
			if exist {
				totalRewards[coin.Denom] = c.Add(coin)
			} else {
				totalRewards[coin.Denom] = coin
			}
		}

		return false
	})

	feePoolRewards := sdk.NewCoins()
	communityPoolRewards := sdk.NewCoins()
	reserveRewards := sdk.NewCoins()

	feePoolRate := params.FeePoolRate.Mul(total)
	communityPoolRate := params.CommunityPoolRate.Mul(total)
	for denom, totalReward := range totalRewards {
		feePoolReward := sdk.NewCoin(denom, feePoolRate.MulInt(totalReward.Amount).RoundInt())
		feePoolRewards = append(feePoolRewards, feePoolReward)

		communityPoolReward := sdk.NewCoin(denom, communityPoolRate.MulInt(totalReward.Amount).RoundInt())
		communityPoolRewards = append(communityPoolRewards, communityPoolReward)

		reserveRewards = append(reserveRewards, sdk.NewCoin(denom, totalReward.Amount.Sub(feePoolReward.Amount).Sub(communityPoolReward.Amount)))
	}

	// fee pool
	if len(feePoolRewards) > 0 {
		err := bk.SendCoinsFromAccountToModule(ctx, rewardDistributeAccount, types.ModuleName, feePoolRewards)
		if err != nil {
			panic(err)
		}

	}
	rewardAccount := k.GetRewardAccount(ctx)
	balances := bk.GetAllBalances(ctx, rewardAccount.GetAddress())
	blockPerYear := k.GetBlocksPerYear(ctx)
	for index, balance := range balances {
		balances[index].Amount = balance.Amount.Quo(sdk.NewInt(int64(blockPerYear)))
	}

	if !balances.IsZero() {
		err := bk.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, balances)
		if err != nil {
			panic(err)
		}
	}

	// community
	if len(communityPoolRewards) > 0 {
		err := dk.FundCommunityPool(ctx, communityPoolRewards, rewardDistributeAccount)
		if err != nil {
			panic(err)
		}
	}

	// reserve
	if len(reserveRewards) > 0 && params.ReserveAccount != "" {
		reserveAccount := sdk.MustAccAddressFromBech32(params.ReserveAccount)
		bk.SendCoins(ctx, rewardDistributeAccount, reserveAccount, reserveRewards)
	}
}
