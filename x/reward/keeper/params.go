package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/reward/types"
)

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k Keeper) GetReserveAccount(ctx sdk.Context) (reserveAccount string) {
	k.paramSpace.Get(ctx, types.ParamStoreKeyReserveAccount, &reserveAccount)
	return reserveAccount
}

func (k Keeper) GetRewardDistributeAccount(ctx sdk.Context) (rewardDistributeAccount string) {
	k.paramSpace.Get(ctx, types.ParamStoreKeyRewardDistributeAccount, &rewardDistributeAccount)
	return rewardDistributeAccount
}
