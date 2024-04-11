package types

const (
	// ModuleName is the module name constant used in many places
	ModuleName = "reward"

	// StoreKey is the store key string for reward
	StoreKey = ModuleName

	// RouterKey is the message route for reward
	RouterKey = ModuleName
)

var (
	ParamsKey = []byte{0x01} // key for reward module params
)
