package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/xpladev/xpla/x/volunteer/client/cli"
)

var (
	ProposalHandler = []govclient.ProposalHandler{
		govclient.NewProposalHandler(cli.GetSubmitProposalRegisterVolunteerValidator),
		govclient.NewProposalHandler(cli.GetSubmitProposalUnregisterVolunteerValidator),
	}
)
