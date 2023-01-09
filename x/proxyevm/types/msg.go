package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
)

const (
	TypeMsgCallEVM = "call_evm"
)

// Route returns the MsgCallEVM message route.
func (msg MsgCallEVM) Route() string { return ModuleName }

// Type returns the MsgCallEVM message type.
func (msg MsgCallEVM) Type() string { return TypeMsgCallEVM }

// GetSigners returns the signer addresses that are expected to sign the result
// of GetSignBytes.
func (msg MsgCallEVM) GetSigners() []sdk.AccAddress {
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{sender}
}

// GetSignBytes returns the raw bytes for a MsgCallEVM message that
// the expected signer needs to sign.
func (msg MsgCallEVM) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic performs basic MsgCallEVM message validation.
func (msg *MsgCallEVM) ValidateBasic() error {
	if msg == nil {
		return sdkerrors.ErrInvalidRequest
	}

	if !msg.Funds.IsValid() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidCoins, msg.Funds.String())
	}
	if _, err := sdk.AccAddressFromBech32(msg.Sender); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Sender)
	}

	if !common.IsHexAddress(msg.Contract) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.Contract)
	}

	return nil
}
