package cli

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func ParseRegisterVolunteerValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.RegisterVolunteerValidatorProposalWithDeposit, error) {
	proposal := types.RegisterVolunteerValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}

func ParseUnregisterVolunteerValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.UnregisterVolunteerValidatorProposalWithDeposit, error) {
	proposal := types.UnregisterVolunteerValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
