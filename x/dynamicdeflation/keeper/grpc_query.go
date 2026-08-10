package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(ctx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	return &types.QueryParamsResponse{Params: params}, nil
}

func (k Keeper) CurrentPeriod(ctx context.Context, req *types.QueryCurrentPeriodRequest) (*types.QueryCurrentPeriodResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	period, err := k.CurrentPeriodStore.Get(ctx)
	if errors.Is(err, collections.ErrNotFound) {
		return &types.QueryCurrentPeriodResponse{}, nil
	}
	if err != nil {
		return nil, err
	}
	return &types.QueryCurrentPeriodResponse{CurrentPeriod: &period}, nil
}

func (k Keeper) Status(ctx context.Context, req *types.QueryStatusRequest) (*types.QueryStatusResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	poolBalance := k.bankKeeper.GetBalance(ctx, k.poolAddress, types.TargetDenom)
	allocated := sdkmath.ZeroInt()
	if has, err := k.CurrentPeriodStore.Has(ctx); err != nil {
		return nil, err
	} else if has {
		period, err := k.CurrentPeriodStore.Get(ctx)
		if err != nil {
			return nil, err
		}
		allocated = period.AllocatedAmount
	}
	surplus, deficit := sdkmath.ZeroInt(), sdkmath.ZeroInt()
	if poolBalance.Amount.GTE(allocated) {
		surplus = poolBalance.Amount.Sub(allocated)
	} else {
		deficit = allocated.Sub(poolBalance.Amount)
	}
	return &types.QueryStatusResponse{
		ModuleBalance:   poolBalance,
		AllocatedAmount: sdk.NewCoin(types.TargetDenom, allocated),
		SurplusAmount:   sdk.NewCoin(types.TargetDenom, surplus),
		DeficitAmount:   sdk.NewCoin(types.TargetDenom, deficit),
	}, nil
}
