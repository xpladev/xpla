package v1_14

import (
	"fmt"

	"cosmossdk.io/collections"
	ccodec "cosmossdk.io/collections/codec"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/xpladev/xpla/app/keepers"
)

// migrateWasmAliases rebuilds from contract metadata rather than trusting the
// legacy auth aliases. Both stores are updated only after the rebuild succeeds.
func migrateWasmAliases(ctx sdk.Context, appKeepers *keepers.AppKeepers) error {
	cacheCtx, write := ctx.CacheContext()
	if err := appKeepers.WasmKeeper.RebuildWasmAliases(cacheCtx); err != nil {
		return fmt.Errorf("rebuild wasm aliases: %w", err)
	}

	// The SDK auth store remains mounted; remove only the retired alias prefix.
	legacyAliases := collections.NewMap(
		collections.NewSchemaBuilder(runtime.NewKVStoreService(appKeepers.GetKVStoreKey()[authtypes.StoreKey])),
		collections.NewPrefix("sliceAddress"),
		"legacySliceAddresses",
		sdk.AccAddressKey,
		ccodec.KeyToValueCodec(sdk.AccAddressKey),
	)
	if err := legacyAliases.Clear(cacheCtx, nil); err != nil {
		return fmt.Errorf("clear legacy auth aliases: %w", err)
	}
	write()
	return nil
}
