package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&RegisterZeroRewardValidatorProposal{}, "xpladev/RegisterZeroRewardValidatorProposal", nil)
	cdc.RegisterConcrete(&UnregisterZeroRewardValidatorProposal{}, "xpladev/UnregisterZeroRewardValidatorProposal", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&RegisterZeroRewardValidatorProposal{},
		&UnregisterZeroRewardValidatorProposal{},
	)
}

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global x/zeroreward module codec. Note, the codec
	// should ONLY be used in certain instances of tests and for JSON encoding as Amino
	// is still used for that purpose.
	//
	// The actual codec used for serialization should be provided to x/zeroreward and
	// defined at the application level.
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
