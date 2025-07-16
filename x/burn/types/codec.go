package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the necessary x/burn interfaces and concrete types
// on the provided LegacyAmino codec. These types are used for Amino JSON serialization.
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgBurn{}, "xpladev/MsgBurn")
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgBurn{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func UnpackMsgBurn(cdc codec.BinaryCodec, msg *types.Any) (*MsgBurn, error) {
	var msgInterface sdk.Msg
	err := cdc.UnpackAny(msg, &msgInterface)
	if err != nil {
		return nil, err
	}

	// Check if it's a MsgBurn
	msgBurn, ok := msgInterface.(*MsgBurn)
	if !ok {
		return nil, sdkerrors.ErrUnpackAny
	}

	return msgBurn, nil
}
