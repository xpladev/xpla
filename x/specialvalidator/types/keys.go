package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "specialvalidator"

	// StoreKey is the store key string for specialvalidator
	StoreKey = ModuleName

	// RouterKey is the message route for specialvalidator
	RouterKey = ModuleName

	// QuerierRoute is the querier route for specialvalidator
	QuerierRoute = ModuleName
)

var (
	// Keys for store prefixes
	SpecialValidatorKey = []byte{0x11}
)

func GetSpecialValidatorKey(operatorAddr sdk.ValAddress) []byte {
	return append(SpecialValidatorKey, address.MustLengthPrefix(operatorAddr)...)
}
