package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgFundXatpPool = "fund_xatp_pool"
)

// NewMsgFundXatpPool returns a new MsgFundXatpPool with a sender and
// a funding amount.
func NewMsgFundXatpPool(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundXatpPool {
	return &MsgFundXatpPool{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// Route returns the MsgFundXatpPool message route.
func (msg MsgFundXatpPool) Route() string { return ModuleName }

// Type returns the MsgFundXatpPool message type.
func (msg MsgFundXatpPool) Type() string { return TypeMsgFundXatpPool }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFundXatpPool) GetSigners() []sdk.AccAddress {
	depoAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{depoAddr}
}

// GetSignBytes returns the raw bytes for a MsgFundXatpPool message that
// the expected signer needs to sign.
func (msg MsgFundXatpPool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundXatpPool message validation.
func (msg MsgFundXatpPool) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if msg.Depositor == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Depositor)
	}

	return nil
}
