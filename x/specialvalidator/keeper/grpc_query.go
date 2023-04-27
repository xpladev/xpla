package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/specialvalidator/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func (k Querier) Specialvalidators(c context.Context, req *types.QuerySpecialValidatorsRequest) (*types.QuerySpecialValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	addresses := k.GetSpecialValidators(ctx)

	specialValidators := []string{}
	for address, _ := range addresses {
		specialValidators = append(specialValidators, address)
	}

	return &types.QuerySpecialValidatorsResponse{SpecialValidators: specialValidators}, nil
}
