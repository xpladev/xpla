package types

import (
	errorsmod "cosmossdk.io/errors"

	"github.com/cosmos/cosmos-sdk/codec/legacy"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgFundRewardPool = "fund_reward_pool"
	TypeMsgUpdateParams   = "update_params"
)

// NewMsgFundRewardPool returns a new MsgFundRewardPool with a sender and
// a funding amount.
func NewMsgFundRewardPool(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundRewardPool {
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
	bz := legacy.Cdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
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

// GetSigners returns the expected signers for a MsgUpdateParams message.
func (msg *MsgUpdateParams) GetSigners() []sdk.AccAddress {
	addr := sdk.MustAccAddressFromBech32(msg.Authority)
	return []sdk.AccAddress{addr}
}

// ValidateBasic does a sanity check of the provided data
func (msg *MsgUpdateParams) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return errorsmod.Wrap(err, "Invalid authority address")
	}

	return msg.Params.ValidateBasic()
}

// GetSignBytes implements the LegacyMsg interface.
func (msg MsgUpdateParams) GetSignBytes() []byte {
	bz := legacy.Cdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}
