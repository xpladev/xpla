package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"

	"github.com/xpladev/xpla/x/burn/types"
)

var _ govtypes.GovHooks = BankGovHooks{}

// BankGovHooks implements govtypes.GovHooks
type BankGovHooks struct {
	keeper     Keeper
	bankKeeper types.BankKeeper
	govKeeper  types.GovKeeper
}

// NewGovHooksForBank creates new gov hooks for bank keeper
func NewGovHooksForBurn(k Keeper, bk types.BankKeeper, gk types.GovKeeper) BankGovHooks {
	return BankGovHooks{keeper: k, bankKeeper: bk, govKeeper: gk}
}

// AfterProposalSubmission implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalSubmission(ctx context.Context, proposalID uint64) error {
	res, err := h.govKeeper.Proposal(ctx, &govv1types.QueryProposalRequest{ProposalId: proposalID})
	if err != nil {
		return err
	}

	proposer, err := sdk.AccAddressFromBech32(res.Proposal.Proposer)
	if err != nil {
		return err
	}

	for _, msg := range res.Proposal.Messages {
		msgBurn, err := types.UnpackMsgBurn(h.keeper.cdc, msg)
		if err != nil {
			// Skip if not MsgBurn
			continue
		}

		burnProposal := types.BurnProposal{
			ProposalId: proposalID,
			Proposer:   proposer.String(),
			Amount:     msgBurn.Amount,
		}

		if err := h.keeper.OngoingBurnProposals.Set(ctx, proposalID, burnProposal); err != nil {
			return err
		}

		if err := h.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, types.ModuleName, msgBurn.Amount); err != nil {
			return err
		}
	}

	return nil
}

// AfterProposalDeposit implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalDeposit(ctx context.Context, proposalID uint64, depositorAddr sdk.AccAddress) error {
	return nil
}

// AfterProposalVote implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalVote(ctx context.Context, proposalID uint64, voterAddr sdk.AccAddress) error {
	return nil
}

// AfterProposalFailedMinDeposit implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalFailedMinDeposit(ctx context.Context, proposalID uint64) error {
	has, err := h.keeper.OngoingBurnProposals.Has(ctx, proposalID)
	if err != nil || !has {
		// Skip if not MsgBurn
		return nil
	}

	burnProposal, err := h.keeper.OngoingBurnProposals.Get(ctx, proposalID)
	if err != nil {
		return err
	}

	proposer, err := sdk.AccAddressFromBech32(burnProposal.Proposer)
	if err != nil {
		return err
	}

	if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, proposer, burnProposal.Amount); err != nil {
		return err
	}

	if err := h.keeper.OngoingBurnProposals.Remove(ctx, proposalID); err != nil {
		return err
	}

	return nil
}

// AfterProposalVotingPeriodEnded implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalVotingPeriodEnded(ctx context.Context, proposalID uint64) error {
	// Check if this is a burn proposal first
	has, err := h.keeper.OngoingBurnProposals.Has(ctx, proposalID)
	if err != nil || !has {
		// Skip if not MsgBurn
		return nil
	}

	// Get proposal details
	res, err := h.govKeeper.Proposal(ctx, &govv1types.QueryProposalRequest{ProposalId: proposalID})
	if err != nil {
		return err
	}

	if err := h.keeper.OngoingBurnProposals.Remove(ctx, proposalID); err != nil {
		return err
	}

	// If proposal passed, burn amount stays in gov module (will be burned)
	if res.Proposal.Status == govv1types.ProposalStatus_PROPOSAL_STATUS_PASSED {
		return nil
	}

	// If proposal failed, return burn amount to proposer
	proposer, err := sdk.AccAddressFromBech32(res.Proposal.Proposer)
	if err != nil {
		return err
	}

	// Find the burn amount from the proposal messages
	for _, msg := range res.Proposal.Messages {
		msgBurn, err := types.UnpackMsgBurn(h.keeper.cdc, msg)
		if err != nil {
			// Skip if not MsgBurn
			continue
		}

		if err := h.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, proposer, msgBurn.Amount); err != nil {
			return err
		}
	}

	return nil
}
