package reward

import (
	"context"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/xpladev/xpla/x/reward/keeper"
	"github.com/xpladev/xpla/x/reward/types"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper, bk types.BankKeeper, sk types.StakingKeeper, dk types.DistributionKeeper) error {
	params, err := k.GetParams(ctx)
	if err != nil {
		return err
	}

	if params.RewardDistributeAccount == "" {
		return nil
	}

	total := params.TotalRate()

	rewardDistributeAccount := sdk.MustAccAddressFromBech32(params.RewardDistributeAccount)

	totalRewards := map[string]sdk.Coin{}

	sk.IterateDelegations(ctx, rewardDistributeAccount, func(index int64, delegation stakingtypes.DelegationI) (stop bool) {
		valAddr, e := sk.ValidatorAddressCodec().StringToBytes(delegation.GetValidatorAddr())
		if e != nil {
			err = e
			return true
		}

		reward, e := dk.WithdrawDelegationRewards(ctx, rewardDistributeAccount, sdk.ValAddress(valAddr))
		if e == disttypes.ErrEmptyDelegationDistInfo {
			return false
		} else if e != nil {
			err = e
			return true
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
	if err != nil {
		return err
	}

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
		err = bk.SendCoinsFromAccountToModule(ctx, rewardDistributeAccount, types.ModuleName, feePoolRewards)
		if err != nil {
			return err
		}

	}
	rewardAccount := k.GetRewardAccount(ctx)
	balances := bk.GetAllBalances(ctx, rewardAccount.GetAddress())
	blockPerYear, err := k.GetBlocksPerYear(ctx)
	if err != nil {
		return err
	}
	for index, balance := range balances {
		balances[index].Amount = balance.Amount.Quo(sdkmath.NewInt(int64(blockPerYear)))
	}

	if !balances.IsZero() {
		err = bk.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, balances)
		if err != nil {
			return err
		}
	}

	// community
	if len(communityPoolRewards) > 0 {
		err = dk.FundCommunityPool(ctx, communityPoolRewards, rewardDistributeAccount)
		if err != nil {
			return err
		}
	}

	// reserve
	if len(reserveRewards) > 0 && params.ReserveAccount != "" {
		reserveAccount := sdk.MustAccAddressFromBech32(params.ReserveAccount)
		bk.SendCoins(ctx, rewardDistributeAccount, reserveAccount, reserveRewards)
	}

	return nil
}
