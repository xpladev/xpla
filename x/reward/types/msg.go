package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgFundRewardPool = "fund_reward_pool"
)

var _ sdk.Msg = &MsgFundRewardPool{}

// NewMsgFundRewardPool returns a new MsgFundRewardPool with a sender and
// a funding amount.
func NewMsgFundRewardPool(amount sdk.Coin, depositor sdk.AccAddress) *MsgFundRewardPool {
	return &MsgFundRewardPool{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// Route returns the MsgFundRewardPool message route.
func (msg MsgFundRewardPool) Route() string { return ModuleName }

// Type returns the MsgFundRewardPool message type.
func (msg MsgFundRewardPool) Type() string { return TypeMsgFundRewardPool }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFundRewardPool) GetSigners() []sdk.AccAddress {
	depoAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{depoAddr}
}

// GetSignBytes returns the raw bytes for a MsgFundRewardPool message that
// the expected signer needs to sign.
func (msg MsgFundRewardPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundRewardPool message validation.
func (msg MsgFundRewardPool) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	if msg.Depositor == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Depositor)
	}

	return nil
}
