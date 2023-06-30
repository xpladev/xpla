package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/xpladev/xpla/x/volunteer/client/cli"
	"github.com/xpladev/xpla/x/volunteer/client/rest"
)

var (
	ProposalHandler = []govclient.ProposalHandler{
		govclient.NewProposalHandler(cli.GetSubmitProposalRegisterVolunteerValidator, rest.RegisterVolunteerValidatorProposalRESTHandler),
		govclient.NewProposalHandler(cli.GetSubmitProposalUnregisterVolunteerValidator, rest.UnregisterVolunteerValidatorProposalRESTHandler),
	}
)
