package cli

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func ParseRegisterSpecialValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.RegisterSpecialValidatorProposalWithDeposit, error) {
	proposal := types.RegisterSpecialValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}

func ParseUnregisterSpecialValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.UnregisterSpecialValidatorProposalWithDeposit, error) {
	proposal := types.UnregisterSpecialValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
