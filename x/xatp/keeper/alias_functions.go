package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func (k Keeper) GetXatpPayerAccount() sdk.AccAddress {
	return k.authKeeper.GetModuleAddress(types.ModuleName)
}

func (k Keeper) DeductAndDistiributeFees(ctx sdk.Context, coins sdk.Coins) error {
	params := k.GetParams(ctx)

	totalRate := params.FeePoolRate.Add(params.CommunityPoolRate).Add(params.RewardPoolRate).Add(params.ReserveRate)

	// fee pool
	feePoolRate := params.FeePoolRate.Quo(totalRate)
	feePoolCoins := sdk.NewCoins()
	for _, coin := range coins {
		feePoolCoin := sdk.NewCoin(coin.Denom, feePoolRate.MulInt(coin.Amount).TruncateInt())
		feePoolCoins = feePoolCoins.Add(feePoolCoin)
	}
	err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, authtypes.FeeCollectorName, feePoolCoins)
	if err != nil {
		return err
	}

	// community pool
	communityPoolRate := params.CommunityPoolRate.Quo(totalRate)
	communityPoolCoins := sdk.NewCoins()
	for _, coin := range coins {
		communityPoolCoin := sdk.NewCoin(coin.Denom, communityPoolRate.MulInt(coin.Amount).TruncateInt())
		communityPoolCoins = communityPoolCoins.Add(communityPoolCoin)

	}

	err = k.distKeeper.FundCommunityPool(ctx, communityPoolCoins, k.GetXatpPayerAccount())
	if err != nil {
		return err
	}

	// reward pool
	rewardPoolRate := params.RewardPoolRate.Quo(totalRate)
	rewardPoolCoins := sdk.NewCoins()
	for _, coin := range coins {
		rewardPoolCoin := sdk.NewCoin(coin.Denom, rewardPoolRate.MulInt(coin.Amount).TruncateInt())
		rewardPoolCoins = rewardPoolCoins.Add(rewardPoolCoin)
	}
	err = k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, rewardtypes.ModuleName, rewardPoolCoins)
	if err != nil {
		return err
	}

	// reserve
	if params.ReserveAccount != "" {
		reserveRate := params.ReserveRate.Quo(totalRate)
		reserveCoins := sdk.NewCoins()
		for _, coin := range coins {
			reserveCoin := sdk.NewCoin(coin.Denom, reserveRate.MulInt(coin.Amount).TruncateInt())
			reserveCoins = reserveCoins.Add(reserveCoin)
		}
		reserveAccount := sdk.MustAccAddressFromBech32(params.ReserveAccount)
		err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, reserveAccount, reserveCoins)
		if err != nil {
			return err
		}
	}

	return nil
}
