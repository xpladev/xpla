package keeper

import (
	"context"

	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func (k Keeper) InitGenesis(ctx context.Context, state *types.GenesisState) error {
	if state == nil {
		state = types.DefaultGenesisState()
	}
	if err := state.Validate(); err != nil {
		return err
	}
	if err := k.SetParams(ctx, state.Params); err != nil {
		return err
	}
	if state.CurrentPeriod != nil {
		if err := k.CurrentPeriodStore.Set(ctx, *state.CurrentPeriod); err != nil {
			return err
		}
	}
	return nil
}

func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	state := &types.GenesisState{Params: params}
	if has, err := k.CurrentPeriodStore.Has(ctx); err != nil {
		return nil, err
	} else if has {
		period, err := k.CurrentPeriodStore.Get(ctx)
		if err != nil {
			return nil, err
		}
		state.CurrentPeriod = &period
	}
	return state, nil
}
