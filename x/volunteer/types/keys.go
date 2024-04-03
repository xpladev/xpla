package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "volunteer"

	// StoreKey is the store key string for volunteer
	StoreKey = ModuleName

	// RouterKey is the message route for volunteer
	RouterKey = ModuleName
)

var (
	// Keys for store prefixes
	VolunteerValidatorKey = []byte{0x11}
)

func GetVolunteerValidatorKey(operatorAddr sdk.ValAddress) []byte {
	return append(VolunteerValidatorKey, address.MustLengthPrefix(operatorAddr)...)
}
