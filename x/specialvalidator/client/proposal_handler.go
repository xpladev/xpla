package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/xpladev/xpla/x/specialvalidator/client/cli"
	"github.com/xpladev/xpla/x/specialvalidator/client/rest"
)

var (
	ProposalHandler = []govclient.ProposalHandler{
		govclient.NewProposalHandler(cli.GetSubmitProposalRegisterSpecialValidator, rest.RegisterSpecialValidatorProposalRESTHandler),
		govclient.NewProposalHandler(cli.GetSubmitProposalUnregisterSpecialValidator, rest.UnregisterSpecialValidatorProposalRESTHandler),
	}
)
