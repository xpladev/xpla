package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&RegisterSpecialValidatorProposal{}, "xpladev/RegisterSpecialvalidatorProposal", nil)
	cdc.RegisterConcrete(&UnregisterSpecialValidatorProposal{}, "xpladev/UnregisterSpecialvalidatorProposal", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&RegisterSpecialValidatorProposal{},
		&UnregisterSpecialValidatorProposal{},
	)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/specialvalidator module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding as Amino
	// is still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/specialvalidator and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
