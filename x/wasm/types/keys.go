package types

import "cosmossdk.io/collections"

// AliasStoreKeyPrefix isolates compatibility aliases from Wasmd's store keys.
var AliasStoreKeyPrefix = collections.NewPrefix("wasmAlias")
