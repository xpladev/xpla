package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/xpladev/xpla/x/burn/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Querier struct {
	Keeper
}

var _ types.QueryServer = Querier{}

func (k Querier) OngoingProposals(c context.Context, req *types.QueryOngoingProposalsRequest) (*types.QueryOngoingProposalsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	proposals, pageRes, err := query.CollectionPaginate(
		c,
		k.OngoingBurnProposals,
		req.Pagination,
		func(_ uint64, proposal types.BurnProposal) (types.BurnProposal, error) {
			return proposal, nil
		},
	)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "paginate: %v", err)
	}

	return &types.QueryOngoingProposalsResponse{Proposals: proposals, Pagination: pageRes}, nil
}

func (k Querier) OngoingProposal(c context.Context, req *types.QueryOngoingProposalRequest) (*types.QueryOngoingProposalResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)
	proposal, err := k.OngoingBurnProposals.Get(ctx, req.ProposalId)
	if err != nil {
		return nil, err
	}

	return &types.QueryOngoingProposalResponse{
		Proposer: proposal.Proposer,
		Amount:   proposal.Amount,
	}, nil
}
