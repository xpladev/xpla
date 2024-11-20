package keeper

import (
	"cosmossdk.io/collections"
	ccodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"

	"github.com/xpladev/xpla/x/auth/types"
)

type AccountKeeper struct {
	authkeeper.AccountKeeper

	addressCodec address.Codec

	cdc          codec.BinaryCodec
	storeService store.KVStoreService

	// State
	SliceAddresses collections.Map[sdk.AccAddress, sdk.AccAddress]
}

func NewAccountKeeper(
	cdc codec.BinaryCodec, storeService store.KVStoreService, proto func() sdk.AccountI,
	maccPerms map[string][]string, ac address.Codec, bech32Prefix, authority string,
) AccountKeeper {

	sb := collections.NewSchemaBuilder(storeService)

	ak := AccountKeeper{
		AccountKeeper:  authkeeper.NewAccountKeeper(cdc, storeService, proto, maccPerms, ac, bech32Prefix, authority),
		addressCodec:   ac,
		cdc:            cdc,
		storeService:   storeService,
		SliceAddresses: collections.NewMap(sb, types.SliceAddressStoreKeyPrefix, "sliceAddresses", sdk.AccAddressKey, ccodec.KeyToValueCodec(sdk.AccAddressKey)),
	}

	return ak
}
