package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	ctx := sdk.UnwrapSDKContext(c)
	proposals := k.GetAllOngoingBurnProposals(ctx)

	return &types.QueryOngoingProposalsResponse{Proposals: proposals}, nil
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
