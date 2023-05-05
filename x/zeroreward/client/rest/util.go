package rest

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type (
	RegisterZeroRewardValidatorProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req,omitempty"`

		Title                string                   `json:"title,omitempty"`
		Description          string                   `json:"description,omitempty"`
		ValidatorDescription stakingtypes.Description `json:"validator_description,omitempty"`
		Pubkey               string                   `json:"pubkey,omitempty"`
		Amount               sdk.Coin                 `json:"amount,omitempty"`
		Proposer             sdk.AccAddress           `json:"proposer,omitempty" yaml:"proposer"`
		Deposit              sdk.Coins                `json:"deposit,omitempty"`
	}

	UnregisterZeroRewardValidatorProposalReq struct {
		BaseReq rest.BaseReq `json:"base_req,omitempty"`

		Title            string         `json:"title,omitempty"`
		Description      string         `json:"description,omitempty"`
		ValidatorAddress sdk.ValAddress `json:"validator_address,omitempty"`
		Proposer         sdk.AccAddress `json:"proposer,omitempty" yaml:"proposer"`
		Deposit          sdk.Coins      `json:"deposit,omitempty"`
	}
)
