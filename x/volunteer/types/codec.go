package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/legacy"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	v1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	legacy.RegisterAminoMsg(cdc, &MsgRegisterVolunteerValidator{}, "xpladev/MsgRegisterVolunteerValidator")
	legacy.RegisterAminoMsg(cdc, &MsgUnregisterVolunteerValidator{}, "xpladev/MsgUnregisterVolunteerValidator")

	cdc.RegisterConcrete(&RegisterVolunteerValidatorProposal{}, "xpladev/RegisterVolunteerValidatorProposal", nil)
	cdc.RegisterConcrete(&RegisterVolunteerValidatorProposalWithDeposit{}, "xpladev/RegisterVolunteerValidatorProposalWithDeposit", nil)
	cdc.RegisterConcrete(&UnregisterVolunteerValidatorProposal{}, "xpladev/UnregisterVolunteerValidatorProposal", nil)
	cdc.RegisterConcrete(&UnregisterVolunteerValidatorProposalWithDeposit{}, "xpladev/UnregisterVolunteerValidatorProposalWithDeposit", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*v1beta1.Content)(nil),
		&RegisterVolunteerValidatorProposal{},
		&UnregisterVolunteerValidatorProposal{},
	)

	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgRegisterVolunteerValidator{},
		&MsgUnregisterVolunteerValidator{},
	)
}
