package client

import (
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	"github.com/xpladev/xpla/x/zeroreward/client/cli"
	"github.com/xpladev/xpla/x/zeroreward/client/rest"
)

var (
	ProposalHandler = []govclient.ProposalHandler{
		govclient.NewProposalHandler(cli.GetSubmitProposalRegisterZeroRewardValidator, rest.RegisterZeroRewardValidatorProposalRESTHandler),
		govclient.NewProposalHandler(cli.GetSubmitProposalUnregisterZeroRewardValidator, rest.UnregisterZeroRewardValidatorProposalRESTHandler),
	}
)
