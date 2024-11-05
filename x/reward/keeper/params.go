package keeper

import (
	"context"

	"github.com/xpladev/xpla/x/reward/types"
)

func (k Keeper) GetParams(ctx context.Context) (params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, _ := store.Get(types.ParamsKey)
	if bz == nil {
		return params
	}

	k.cdc.MustUnmarshal(bz, &params)
	return params
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.ValidateBasic(); err != nil {
		return err
	}

	store := k.storeService.OpenKVStore(ctx)
	bz, err := k.cdc.Marshal(&params)
	if err != nil {
		return err
	}

	return store.Set(types.ParamsKey, bz)
}

func (k Keeper) GetReserveAccount(ctx context.Context) (reserveAccount string) {
	return k.GetParams(ctx).ReserveAccount
}

func (k Keeper) GetRewardDistributeAccount(ctx context.Context) (rewardDistributeAccount string) {
	return k.GetParams(ctx).RewardDistributeAccount
}
