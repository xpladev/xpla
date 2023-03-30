package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/xpladev/xpla/x/xatp/client/cli"
	"github.com/xpladev/xpla/x/xatp/client/rest"
)

var (
	ProposalHandler = []govclient.ProposalHandler{
		govclient.NewProposalHandler(cli.GetSubmitProposalRegisterXatp, rest.RegisterXatpProposalRESTHandler),
		govclient.NewProposalHandler(cli.GetSubmitProposalUnregisterXatp, rest.UnregisterXatpProposalRESTHandler),
	}
)
