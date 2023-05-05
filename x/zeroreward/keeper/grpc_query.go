package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/zeroreward/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func (k Querier) ZeroRewardValidators(c context.Context, req *types.QueryZeroRewardValidatorsRequest) (*types.QueryZeroRewardValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	addresses := k.GetZeroRewardValidators(ctx)

	zeroRewardValidators := []string{}
	for address, _ := range addresses {
		zeroRewardValidators = append(zeroRewardValidators, address)
	}

	return &types.QueryZeroRewardValidatorsResponse{ZeroRewardValidators: zeroRewardValidators}, nil
}
