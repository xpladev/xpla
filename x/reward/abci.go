package reward

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	disttypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/xpladev/xpla/x/reward/keeper"
	"github.com/xpladev/xpla/x/reward/types"
)

func EndBlocker(ctx sdk.Context, req abci.RequestEndBlock, k keeper.Keeper, bk types.BankKeeper, dk types.DistributionKeeper) []abci.ValidatorUpdate {
	params := k.GetParams(ctx)
	total := params.TotalRate()

	totalRewards := map[string]sdk.Coin{}
	for _, strValidator := range params.Validators {
		validator, err := sdk.ValAddressFromBech32(strValidator)
		if err != nil {
			panic(err)
		}
		reward, err := dk.WithdrawDelegationRewards(ctx, types.DelegateProxyAccount, validator)
		if err == disttypes.ErrEmptyDelegationDistInfo {
			continue
		} else if err != nil {
			panic(err)
		}

		for _, coin := range reward {
			c, exist := totalRewards[coin.Denom]
			if exist {
				totalRewards[coin.Denom] = c.Add(coin)
			} else {
				totalRewards[coin.Denom] = coin
			}
		}
	}

	feePoolRewards := sdk.NewCoins()
	communityPoolRewards := sdk.NewCoins()
	reserveRewards := sdk.NewCoins()

	feePoolRate := params.FeePoolRate.Mul(total)
	communityPoolRate := params.CommunityPoolRate.Mul(total)
	reserveRate := params.ReserveRate.Mul(total)
	for denom, totalReward := range totalRewards {
		feePoolRewards = append(feePoolRewards, sdk.NewCoin(denom, feePoolRate.MulInt(totalReward.Amount).RoundInt()))
		communityPoolRewards = append(communityPoolRewards, sdk.NewCoin(denom, communityPoolRate.MulInt(totalReward.Amount).RoundInt()))
		reserveRewards = append(reserveRewards, sdk.NewCoin(denom, reserveRate.MulInt(totalReward.Amount).RoundInt()))
	}

	// fee pool
	if len(feePoolRewards) > 0 {
		err := bk.SendCoinsFromAccountToModule(ctx, types.DelegateProxyAccount, disttypes.ModuleName, feePoolRewards)
		if err != nil {
			panic(err)
		}
	}

	// community
	if len(communityPoolRewards) > 0 {
		err := dk.FundCommunityPool(ctx, communityPoolRewards, types.DelegateProxyAccount)
		if err != nil {
			panic(err)
		}
	}

	// reserve
	if len(reserveRewards) > 0 && params.ReserveAccount != "" {
		reserveAccount := sdk.MustAccAddressFromBech32(params.ReserveAccount)
		bk.SendCoins(ctx, types.DelegateProxyAccount, reserveAccount, reserveRewards)
	}

	return []abci.ValidatorUpdate{}
}
