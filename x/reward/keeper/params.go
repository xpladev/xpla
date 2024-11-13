package keeper

import (
	"context"

	"github.com/xpladev/xpla/x/reward/types"
)

func (k Keeper) GetParams(ctx context.Context) (params types.Params, err error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil {
		return params, err
	}

	if bz == nil {
		return params, nil
	}

	err = k.cdc.Unmarshal(bz, &params)
	return params, err
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
