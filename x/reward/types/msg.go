package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	TypeMsgFundFeeCollector = "fund_fee_collector"
)

// NewMsgFundFeeCollector returns a new MsgFundFeeCollector with a sender and
// a funding amount.
func NewMsgFundFeeCollector(amount sdk.Coins, depositor sdk.AccAddress) *MsgFundFeeCollector {
	return &MsgFundFeeCollector{
		Amount:    amount,
		Depositor: depositor.String(),
	}
}

// Route returns the MsgFundFeeCollector message route.
func (msg MsgFundFeeCollector) Route() string { return ModuleName }

// Type returns the MsgFundFeeCollector message type.
func (msg MsgFundFeeCollector) Type() string { return TypeMsgFundFeeCollector }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgFundFeeCollector) GetSigners() []sdk.AccAddress {
	depoAddr, err := sdk.AccAddressFromBech32(msg.Depositor)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{depoAddr}
}

// GetSignBytes returns the raw bytes for a MsgFundFeeCollector message that
// the expected signer needs to sign.
func (msg MsgFundFeeCollector) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgFundFeeCollector message validation.
func (msg MsgFundFeeCollector) ValidateBasic() error {
	if !msg.Amount.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}
	if msg.Depositor == "" {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Depositor)
	}

	return nil
}
