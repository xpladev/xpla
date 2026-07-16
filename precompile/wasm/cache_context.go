package wasm

import (
	"io"
	"sort"

	"cosmossdk.io/store/cachekv"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// cacheNativeWasmContext isolates wasm's nested CacheContext writes until the
// native action succeeds. Its commit writes into the current StateDB snapshot
// layer without calling snapshotmulti.Store.Write, so a later EVM frame revert
// can still discard the wasm state through the StateDB journal.
func cacheNativeWasmContext(ctx sdk.Context) (sdk.Context, func()) {
	cache := newIsolatedCacheMultiStore(ctx.MultiStore())
	cacheCtx := ctx.WithMultiStore(cache).WithEventManager(sdk.NewEventManager())

	writeCache := func() {
		ctx.EventManager().EmitEvents(cacheCtx.EventManager().Events())
		cache.Write()
	}
	return cacheCtx, writeCache
}

// isolatedCacheMultiStore lazily branches stores and only writes the branch
// into its immediate parent. In particular, Write never calls parent.Write.
type isolatedCacheMultiStore struct {
	parent storetypes.MultiStore
	stores map[storetypes.StoreKey]*cachekv.Store
}

var _ storetypes.CacheMultiStore = (*isolatedCacheMultiStore)(nil)

func newIsolatedCacheMultiStore(parent storetypes.MultiStore) *isolatedCacheMultiStore {
	return &isolatedCacheMultiStore{
		parent: parent,
		stores: make(map[storetypes.StoreKey]*cachekv.Store),
	}
}

func (s *isolatedCacheMultiStore) GetStoreType() storetypes.StoreType {
	return storetypes.StoreTypeMulti
}

func (s *isolatedCacheMultiStore) CacheWrap() storetypes.CacheWrap {
	return s.CacheMultiStore().(storetypes.CacheWrap)
}

func (s *isolatedCacheMultiStore) CacheWrapWithTrace(io.Writer, storetypes.TraceContext) storetypes.CacheWrap {
	return s.CacheWrap()
}

func (s *isolatedCacheMultiStore) CacheMultiStore() storetypes.CacheMultiStore {
	return newIsolatedCacheMultiStore(s)
}

func (s *isolatedCacheMultiStore) CacheMultiStoreWithVersion(version int64) (storetypes.CacheMultiStore, error) {
	parent, err := s.parent.CacheMultiStoreWithVersion(version)
	if err != nil {
		return nil, err
	}
	return newIsolatedCacheMultiStore(parent), nil
}

func (s *isolatedCacheMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	return s.getStore(key)
}

func (s *isolatedCacheMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	return s.getStore(key)
}

func (s *isolatedCacheMultiStore) getStore(key storetypes.StoreKey) *cachekv.Store {
	if store, ok := s.stores[key]; ok {
		return store
	}

	store := cachekv.NewStore(s.parent.GetKVStore(key))
	s.stores[key] = store
	return store
}

func (s *isolatedCacheMultiStore) TracingEnabled() bool {
	return s.parent.TracingEnabled()
}

func (s *isolatedCacheMultiStore) SetTracer(io.Writer) storetypes.MultiStore {
	return s
}

func (s *isolatedCacheMultiStore) SetTracingContext(storetypes.TraceContext) storetypes.MultiStore {
	return s
}

func (s *isolatedCacheMultiStore) LatestVersion() int64 {
	return s.parent.LatestVersion()
}

func (s *isolatedCacheMultiStore) Write() {
	keys := make([]storetypes.StoreKey, 0, len(s.stores))
	for key := range s.stores {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		return keys[i].Name() < keys[j].Name()
	})

	for _, key := range keys {
		s.stores[key].Write()
	}
}
