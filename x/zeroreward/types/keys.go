package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
)

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "zeroreward"

	// StoreKey is the store key string for zeroreward
	StoreKey = ModuleName

	// RouterKey is the message route for zeroreward
	RouterKey = ModuleName

	// QuerierRoute is the querier route for zeroreward
	QuerierRoute = ModuleName
)

var (
	// Keys for store prefixes
	ZeroRewardValidatorKey = []byte{0x11}
)

func GetZeroRewardValidatorKey(operatorAddr sdk.ValAddress) []byte {
	return append(ZeroRewardValidatorKey, address.MustLengthPrefix(operatorAddr)...)
}
