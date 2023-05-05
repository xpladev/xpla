package cli

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func ParseRegisterZeroRewardValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.RegisterZeroRewardValidatorProposalWithDeposit, error) {
	proposal := types.RegisterZeroRewardValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}

func ParseUnregisterZeroRewardValidatorProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.UnregisterZeroRewardValidatorProposalWithDeposit, error) {
	proposal := types.UnregisterZeroRewardValidatorProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
