package types

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgFundRewardPool = "fund_reward_pool"
	TypeMsgUpdateParams   = "update_params"
)

var (
	_ sdk.Msg = (*MsgFundRewardPool)(nil)
	_ sdk.Msg = (*MsgUpdateParams)(nil)
)

// NewMsgFundRewardPool returns a new MsgFundRewardPool with a sender and
// a funding amount.
func NewMsgFundRewardPool(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundRewardPool {
	return &MsgFundRewardPool{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// ValidateBasic performs basic MsgFundRewardPool message validation.
func (msg MsgFundRewardPool) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if msg.Depositor == "" {
		return errorsmod.Wrap(sdkerrors.ErrInvalidAddress, msg.Depositor)
	}

	return nil
}

// ValidateBasic does a sanity check of the provided data
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "Invalid authority address")
	}

	return msg.Params.ValidateBasic()
}
