package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/xpladev/xpla/x/bank/types"
)

var _ govtypes.GovHooks = BankGovHooks{}

// GovHooks implements govtypes.GovHooks
type BankGovHooks struct {
	bankKeeper Keeper
	govKeeper  types.GovKeeper
}

// NewGovHooks creates new gov hooks for bank keeper
func NewGovHooksForBank(bk Keeper, gk types.GovKeeper) BankGovHooks {
	return BankGovHooks{bankKeeper: bk, govKeeper: gk}
}

// AfterProposalSubmission implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalSubmission(ctx context.Context, proposalID uint64) error {
	res, err := h.govKeeper.Proposal(ctx, &govv1types.QueryProposalRequest{ProposalId: proposalID})
	if err != nil {
		return err
	}

	for _, msg := range res.Proposal.Messages {
		msgBurn, err := types.UnpackMsgBurn(h.bankKeeper.cdc, msg)
		if err != nil {
			// only execute when MsgBurn
			continue
		}

		proposer, err := sdk.AccAddressFromBech32(res.Proposal.Proposer)
		if err != nil {
			return err
		}

		err = h.bankKeeper.SendCoinsFromAccountToModule(ctx, proposer, govtypes.ModuleName, msgBurn.Amount)
		if err != nil {
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
	return nil
}

// AfterProposalVotingPeriodEnded implements govtypes.GovHooks
func (h BankGovHooks) AfterProposalVotingPeriodEnded(ctx context.Context, proposalID uint64) error {
	res, err := h.govKeeper.Proposal(ctx, &govv1types.QueryProposalRequest{ProposalId: proposalID})
	if err != nil {
		return err
	}

	// Only process if proposal was rejected (status is ProposalStatus_PROPOSAL_STATUS_PASSED)
	if res.Proposal.Status == govv1types.ProposalStatus_PROPOSAL_STATUS_PASSED {
		return nil
	}

	for _, msg := range res.Proposal.Messages {
		msgBurn, err := types.UnpackMsgBurn(h.bankKeeper.cdc, msg)
		if err != nil {
			// only execute when MsgBurn
			continue
		}

		proposer, err := sdk.AccAddressFromBech32(res.Proposal.Proposer)
		if err != nil {
			return err
		}

		err = h.bankKeeper.SendCoinsFromModuleToAccount(ctx, govtypes.ModuleName, proposer, msgBurn.Amount)
		if err != nil {
			return err
		}
	}

	return nil
}
