package keeper

import (
	"context"

	"github.com/xpladev/xpla/x/burn/types"
)

// InitGenesis initializes the bank module's state from a given genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState *types.GenesisState) {
	for _, proposal := range genState.OngoingBurnProposals {
		k.OngoingBurnProposals.Set(ctx, proposal.ProposalId, proposal)
	}
}

// ExportGenesis returns the bank module's genesis state.
func (k Keeper) ExportGenesis(ctx context.Context) *types.GenesisState {
	rv := types.NewGenesisState(
		k.GetAllOngoingBurnProposals(ctx),
	)
	return rv
}
