package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BurnProposalSlice is an alias for []*BurnProposal
type BurnProposalSlice = []*BurnProposal

func (b BurnProposal) Validate() error {
	if b.ProposalId == 0 {
		return errors.New("proposal ID cannot be 0")
	}

	if _, err := sdk.AccAddressFromBech32(b.Proposer); err != nil {
		return err
	}

	if err := b.Amount.Validate(); err != nil {
		return err
	}

	return nil
}
