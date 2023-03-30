package cli

import (
	"os"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/xpladev/xpla/x/xatp/types"
)

func ParseRegisterXatpProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.RegisterXatpProposalWithDeposit, error) {
	proposal := types.RegisterXatpProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}

func ParseUnregisterXatpProposalWithDeposit(cdc codec.JSONCodec, proposalFile string) (types.UnregisterXatpProposalWithDeposit, error) {
	proposal := types.UnregisterXatpProposalWithDeposit{}

	contents, err := os.ReadFile(proposalFile)
	if err != nil {
		return proposal, err
	}

	if err = cdc.UnmarshalJSON(contents, &proposal); err != nil {
		return proposal, err
	}

	return proposal, nil
}
