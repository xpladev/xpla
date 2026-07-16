package wasm

import (
	"testing"

	"github.com/stretchr/testify/require"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/evm/x/vm/store/snapshotmulti"

	"cosmossdk.io/store/cachekv"
	"cosmossdk.io/store/dbadapter"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func TestCacheNativeWasmContextKeepsWritesInsideStateDBSnapshot(t *testing.T) {
	snapshotStore, baseStore, key := setupSnapshotContext(t)
	snapshot := snapshotStore.Snapshot()
	ctx := sdk.Context{}.
		WithMultiStore(snapshotStore).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmCtx, writeWasm := cacheNativeWasmContext(ctx)
	wasmCtx.KVStore(key).Set([]byte("counter"), []byte("1"))

	subCtx, writeSubmessage := wasmCtx.CacheContext()
	subCtx.KVStore(key).Set([]byte("reply"), []byte("2"))
	writeSubmessage()
	writeWasm()

	require.Equal(t, []byte("1"), snapshotStore.GetKVStore(key).Get([]byte("counter")))
	require.Equal(t, []byte("2"), snapshotStore.GetKVStore(key).Get([]byte("reply")))
	require.Nil(t, baseStore.Get([]byte("counter")))
	require.Nil(t, baseStore.Get([]byte("reply")))

	snapshotStore.RevertToSnapshot(snapshot)
	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("counter")))
	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("reply")))
}

func TestCacheNativeWasmContextDiscardsFailedAction(t *testing.T) {
	snapshotStore, _, key := setupSnapshotContext(t)
	snapshotStore.Snapshot()
	ctx := sdk.Context{}.
		WithMultiStore(snapshotStore).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewInfiniteGasMeter())

	wasmCtx, _ := cacheNativeWasmContext(ctx)
	wasmCtx.KVStore(key).Set([]byte("counter"), []byte("1"))

	require.Nil(t, snapshotStore.GetKVStore(key).Get([]byte("counter")))
}

func setupSnapshotContext(t *testing.T) (*snapshotmulti.Store, *cachekv.Store, *storetypes.KVStoreKey) {
	t.Helper()

	key := storetypes.NewKVStoreKey("wasm")
	baseStore := cachekv.NewStore(dbadapter.Store{DB: dbm.NewMemDB()})
	stores := map[*storetypes.KVStoreKey]storetypes.CacheWrap{key: baseStore}
	return snapshotmulti.NewStoreWithKVStores(stores), baseStore, key
}
