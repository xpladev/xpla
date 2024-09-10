package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func (k Querier) VolunteerValidators(c context.Context, req *types.QueryVolunteerValidatorsRequest) (*types.QueryVolunteerValidatorsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	addresses, err := k.GetVolunteerValidators(ctx)
	if err != nil {
		return nil, err
	}

	volunteerValidators := []string{}
	for address, _ := range addresses {
		volunteerValidators = append(volunteerValidators, address)
	}

	return &types.QueryVolunteerValidatorsResponse{VolunteerValidators: volunteerValidators}, nil
}
