package keepers

import (
	"fmt"
	"time"

	"cosmossdk.io/collections"
	collectionscodec "cosmossdk.io/collections/codec"
	"cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// newLegacyCompatibleFeeAllowanceQueue preserves the queue key layout used by
// feegrant before Cosmos SDK v0.54 while accepting its empty presence-marker
// value. New entries continue to use the canonical collections bool encoding.
func newLegacyCompatibleFeeAllowanceQueue(
	storeService store.KVStoreService,
	canonicalQueue collections.Map[collections.Triple[time.Time, sdk.AccAddress, sdk.AccAddress], bool],
) collections.Map[collections.Triple[time.Time, sdk.AccAddress, sdk.AccAddress], bool] {
	sb := collections.NewSchemaBuilder(storeService)
	valueCodec := collectionscodec.NewAltValueCodec(
		collections.BoolValue,
		func(value []byte) (bool, error) {
			if len(value) == 0 {
				return true, nil
			}

			return false, fmt.Errorf(
				"%w: invalid legacy feegrant expiration queue value: %x",
				collections.ErrEncoding,
				value,
			)
		},
	)

	return collections.NewMap(
		sb,
		collections.NewPrefix(canonicalQueue.GetPrefix()),
		canonicalQueue.GetName(),
		canonicalQueue.KeyCodec(),
		valueCodec,
	)
}
