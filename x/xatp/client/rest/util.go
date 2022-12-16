package rest

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/xpladev/xpla/x/xatp/types"
)

type (
	RegisterXatpProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req,omitempty"`

		Title       string         `json:"title,omitempty"`
		Description string         `json:"description,omitempty"`
		Xatp        types.XATP     `json:"xatp,omitempty"`
		Proposer    sdk.AccAddress `json:"proposer" yaml:"proposer"`
		Deposit     sdk.Coins      `json:"deposit,omitempty"`
	}

	UnregisterXatpProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req,omitempty"`

		Title       string         `json:"title,omitempty"`
		Description string         `json:"description,omitempty"`
		Denom       string         `json:"denom,omitempty"`
		Proposer    sdk.AccAddress `json:"proposer" yaml:"proposer"`
		Deposit     sdk.Coins      `json:"deposit,omitempty"`
	}
)
