package keeper

import (
	"context"

	"cosmossdk.io/store/prefix"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/query"
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

	volunteerValidators := []string{}
	store := prefix.NewStore(runtime.KVStoreAdapter(k.storeService.OpenKVStore(c)), types.VolunteerValidatorKey)
	pageRes, err := query.Paginate(store, req.Pagination, func(_, value []byte) error {
		validator := types.VolunteerValidator{}
		if err := k.cdc.Unmarshal(value, &validator); err != nil {
			return err
		}

		volunteerValidators = append(volunteerValidators, validator.Address)
		return nil
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	return &types.QueryVolunteerValidatorsResponse{
		VolunteerValidators: volunteerValidators,
		Pagination:          pageRes,
	}, nil
}
