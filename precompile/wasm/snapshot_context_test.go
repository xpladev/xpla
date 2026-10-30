package wasm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/dbadapter"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/store/snapshotmulti"
)

func TestWasmCacheContextKeepsWritesInsideStateDBSnapshot(t *testing.T) {
	snapshotStore, baseStore, key := setupSnapshotContext(t)
	snapshot := snapshotStore.Snapshot()
	ctx := sdk.Context{}.
		WithMultiStore(snapshotStore).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmCtx, writeWasm := ctx.CacheContext()
	require.Same(t, ctx.GasMeter(), wasmCtx.GasMeter())
	wasmCtx.KVStore(key).Set([]byte("counter"), []byte("1"))
	wasmCtx.EventManager().EmitEvent(sdk.NewEvent("wasm_execute"))
	require.Empty(t, ctx.EventManager().Events())

	subCtx, writeSubmessage := wasmCtx.CacheContext()
	retained := subCtx.MultiStore().GetKVStore(key).CacheWrap()
	subCtx.KVStore(key).Set([]byte("reply"), []byte("2"))
	writeSubmessage()
	require.Panics(t, retained.Write)
	writeWasm()
	require.Equal(t, sdk.Events{sdk.NewEvent("wasm_execute")}, ctx.EventManager().Events())
	require.Panics(t, func() { wasmCtx.KVStore(key).Get([]byte("counter")) })

	require.Equal(t, []byte("1"), snapshotStore.GetKVStore(key).Get([]byte("counter")))
	require.Equal(t, []byte("2"), snapshotStore.GetKVStore(key).Get([]byte("reply")))
	require.Nil(t, baseStore.Get([]byte("counter")))
	require.Nil(t, baseStore.Get([]byte("reply")))

	snapshotStore.RevertToSnapshot(snapshot)
	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("counter")))
	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("reply")))
}

func TestWasmCacheContextDiscardsFailedAction(t *testing.T) {
	snapshotStore, _, key := setupSnapshotContext(t)
	snapshot := snapshotStore.Snapshot()
	ctx := sdk.Context{}.
		WithMultiStore(snapshotStore).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmCtx, _ := ctx.CacheContext()
	wasmCtx.KVStore(key).Set([]byte("counter"), []byte("1"))
	wasmCtx.EventManager().EmitEvent(sdk.NewEvent("wasm_execute"))
	retained := wasmCtx.MultiStore().GetKVStore(key).CacheWrap().(storetypes.CacheKVStore)
	retained.Set([]byte("stale"), []byte("value"))

	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("counter")))
	require.Empty(t, ctx.EventManager().Events())

	snapshotStore.RevertToSnapshot(snapshot)
	snapshotStore.Snapshot()
	require.Panics(t, retained.Write)
	require.Panics(t, wasmCtx.MultiStore().(storetypes.CacheMultiStore).Write)
	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("stale")))
	require.Empty(t, ctx.EventManager().Events())
}

func setupSnapshotContext(t *testing.T) (*snapshotmulti.Store, *cachekv.Store, *storetypes.KVStoreKey) {
	t.Helper()

	key := storetypes.NewKVStoreKey("wasm")
	baseStore := cachekv.NewStore(dbadapter.Store{DB: dbm.NewMemDB()})
	stores := map[*storetypes.KVStoreKey]storetypes.CacheWrap{key: baseStore}
	return snapshotmulti.NewStoreWithKVStores(stores), baseStore, key
}
