package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

// Params queries params of distribution module
func (k Querier) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	ctx := sdk.UnwrapSDKContext(c)
	var params types.Params
	k.paramSpace.GetParamSet(ctx, &params)

	return &types.QueryParamsResponse{Params: params}, nil
}

func (k Querier) Xatps(c context.Context, req *types.QueryXatpsRequest) (*types.QueryXatpsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	xatps := k.GetAllXatps(ctx)

	return &types.QueryXatpsResponse{Xatps: xatps}, nil
}

func (k Querier) Xatp(c context.Context, req *types.QueryXatpRequest) (*types.QueryXatpResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	xatp, found := k.GetXatp(ctx, req.Denom)
	if !found {
		return nil, status.Error(codes.NotFound, req.Denom)
	}

	return &types.QueryXatpResponse{Xatp: xatp}, nil
}
