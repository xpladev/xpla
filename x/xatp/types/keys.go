package types

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "xatp"

	// StoreKey is the store key string for xatp
	StoreKey = ModuleName

	// RouterKey is the message route for xatp
	RouterKey = ModuleName

	// QuerierRoute is the querier route for xatp
	QuerierRoute = ModuleName
)

var (
	// Keys for store prefixes
	XatpsKey = []byte{0x11}
)

func GetXatpKey(denom string) []byte {
	return append(XatpsKey, []byte(denom)...)
}
