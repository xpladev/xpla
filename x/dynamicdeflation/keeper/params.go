package keeper

import (
	"context"

	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func (k Keeper) GetParams(ctx context.Context) (types.Params, error) {
	return k.ParamsStore.Get(ctx)
}

func (k Keeper) SetParams(ctx context.Context, params types.Params) error {
	if err := params.Validate(); err != nil {
		return err
	}
	return k.ParamsStore.Set(ctx, params)
}
