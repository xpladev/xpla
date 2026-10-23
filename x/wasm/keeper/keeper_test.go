package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	upstream "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/wasm/types"

	corestore "cosmossdk.io/core/store"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
)

func TestAliasRegistryUsesOnlyWasmStore(t *testing.T) {
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	wasmKey := storetypes.NewKVStoreKey("wasm")
	ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{authtypes.StoreKey: authKey, "wasm": wasmKey}, nil, nil)
	accountKeeper := newTestAccountKeeperWithService(t, runtime.NewKVStoreService(authKey), runtime.NewKVStoreService(wasmKey))
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	contract := sdk.AccAddress(append(bytes.Repeat([]byte{0x32}, 12), alias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, contract, 7))
	snapshot := func(key *storetypes.KVStoreKey) map[string]string {
		result := make(map[string]string)
		iterator := ctx.KVStore(key).Iterator(nil, nil)
		defer func() { require.NoError(t, iterator.Close()) }()
		for ; iterator.Valid(); iterator.Next() {
			result[string(iterator.Key())] = string(iterator.Value())
		}
		return result
	}
	authBefore := snapshot(authKey)
	require.Empty(t, snapshot(wasmKey))
	require.NoError(t, accountKeeper.registry.RegisterWasmAlias(ctx, alias, contract))
	require.Equal(t, authBefore, snapshot(authKey), "registration must not write to auth")
	wasmEntries := snapshot(wasmKey)
	require.Len(t, wasmEntries, 1)
	for key := range wasmEntries {
		require.True(t, bytes.HasPrefix([]byte(key), types.AliasStoreKeyPrefix.Bytes()))
	}
	require.Nil(t, accountKeeper.GetAccount(ctx, alias), "SDK account lookup stays exact")
	require.False(t, accountKeeper.HasAccount(ctx, alias))
	require.True(t, accountKeeper.HasAccount(ctx, contract))
	require.NoError(t, accountKeeper.registry.RebuildWasmAliases(ctx))
	require.Empty(t, snapshot(wasmKey))
	require.Equal(t, authBefore, snapshot(authKey), "rebuild must not write to auth")
}

func TestWasmAliasRegistrationAndAliasPrecedence(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20))
	contract := sdk.AccAddress(append(bytes.Repeat([]byte{0x44}, 12), alias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, contract, 7))

	require.NoError(t, accountKeeper.registry.RegisterWasmAlias(ctx, alias, contract))
	require.NoError(t, accountKeeper.registry.RegisterWasmAlias(ctx, alias, contract), "same pair must be idempotent")
	resolved, found, err := accountKeeper.registry.ResolveWasmAlias(ctx, alias)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, contract, resolved)

	exact := newBaseAccount(t, accountKeeper, ctx, alias, 18)
	accountKeeper.SetAccount(ctx, exact)
	resolved, found, err = accountKeeper.registry.ResolveWasmAlias(ctx, alias)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, contract, resolved, "explicit Wasm resolution remains separate")
}

func TestRegisterWasmAliasRejectsInvalidMappings(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x55}, 20))
	contract := sdk.AccAddress(append(bytes.Repeat([]byte{0x66}, 12), alias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, contract, 0))

	cases := map[string]struct {
		alias    sdk.AccAddress
		contract sdk.AccAddress
	}{
		"short key":       {sdk.AccAddress(bytes.Repeat([]byte{1}, 19)), contract},
		"short contract":  {alias, sdk.AccAddress(bytes.Repeat([]byte{2}, 31))},
		"suffix mismatch": {sdk.AccAddress(bytes.Repeat([]byte{3}, 20)), contract},
		"missing account": {sdk.AccAddress(bytes.Repeat([]byte{4}, 20)), sdk.AccAddress(bytes.Repeat([]byte{4}, 32))},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			require.Error(t, accountKeeper.registry.RegisterWasmAlias(ctx, tc.alias, tc.contract))
		})
	}

	conflict := sdk.AccAddress(append(bytes.Repeat([]byte{0x77}, 12), alias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, conflict, 0))
	require.NoError(t, accountKeeper.registry.RegisterWasmAlias(ctx, alias, contract))
	require.ErrorContains(t, accountKeeper.registry.RegisterWasmAlias(ctx, alias, conflict), "already resolves")

	for _, moduleName := range []string{govtypes.ModuleName, minttypes.ModuleName} {
		moduleAlias := authtypes.NewModuleAddress(moduleName)
		moduleContract := sdk.AccAddress(append(bytes.Repeat([]byte{0x88}, 12), moduleAlias...))
		accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, moduleContract, 0))
		require.ErrorContains(t, accountKeeper.registry.RegisterWasmAlias(ctx, moduleAlias, moduleContract), "protected module")
		require.NoError(t, accountKeeper.registry.aliases.Set(ctx, moduleAlias, moduleContract))
		_, _, err := accountKeeper.registry.ResolveWasmAlias(ctx, moduleAlias)
		require.ErrorContains(t, err, "protected module")
	}
}

func TestResolveWasmAliasRejectsMalformedLegacyState(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x91}, 20))
	wrongSuffix := sdk.AccAddress(bytes.Repeat([]byte{0x92}, 32))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, wrongSuffix, 0))
	require.NoError(t, accountKeeper.registry.aliases.Set(ctx, alias, wrongSuffix))
	_, _, err := accountKeeper.registry.ResolveWasmAlias(ctx, alias)
	require.ErrorContains(t, err, "does not match")

	require.NoError(t, accountKeeper.registry.aliases.Set(ctx, alias, sdk.AccAddress(bytes.Repeat([]byte{0x93}, 31))))
	_, _, err = accountKeeper.registry.ResolveWasmAlias(ctx, alias)
	require.ErrorContains(t, err, "must be 32 bytes")
}

func TestRebuildWasmAliasesIsDeterministicAndAtomic(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	aliasA := sdk.AccAddress(bytes.Repeat([]byte{0xa1}, 20))
	aliasB := sdk.AccAddress(bytes.Repeat([]byte{0xb1}, 20))
	contractA := sdk.AccAddress(append(bytes.Repeat([]byte{0xa2}, 12), aliasA...))
	contractB := sdk.AccAddress(append(bytes.Repeat([]byte{0xb2}, 12), aliasB...))
	for _, contract := range []sdk.AccAddress{contractA, contractB} {
		accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, contract, 0))
	}
	legacyAlias := sdk.AccAddress(bytes.Repeat([]byte{0xc1}, 20))
	legacyTarget := sdk.AccAddress(bytes.Repeat([]byte{0xc2}, 32))
	require.NoError(t, accountKeeper.registry.aliases.Set(ctx, legacyAlias, legacyTarget))

	setContractMetadata(t, accountKeeper, ctx, contractB, contractA)
	require.NoError(t, accountKeeper.registry.RebuildWasmAliases(ctx))
	require.Equal(t, map[string]string{
		string(aliasA): string(contractA),
		string(aliasB): string(contractB),
	}, snapshotAliases(t, accountKeeper, ctx))

	setContractMetadata(t, accountKeeper, ctx, contractA, contractB)
	require.NoError(t, accountKeeper.registry.RebuildWasmAliases(ctx))
	expected := snapshotAliases(t, accountKeeper, ctx)

	conflict := sdk.AccAddress(append(bytes.Repeat([]byte{0xd2}, 12), aliasA...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, conflict, 0))
	var conflictErrors []string
	for _, candidates := range [][]sdk.AccAddress{{contractA, conflict}, {conflict, contractA}} {
		setContractMetadata(t, accountKeeper, ctx, candidates...)
		err := accountKeeper.registry.RebuildWasmAliases(ctx)
		require.ErrorContains(t, err, "share alias")
		conflictErrors = append(conflictErrors, err.Error())
		require.Equal(t, expected, snapshotAliases(t, accountKeeper, ctx), "validation failure must preserve prior registry")
	}
	require.Equal(t, conflictErrors[0], conflictErrors[1], "collision failure must not depend on input order")

	moduleAlias := authtypes.NewModuleAddress(govtypes.ModuleName)
	moduleContract := sdk.AccAddress(append(bytes.Repeat([]byte{0xe2}, 12), moduleAlias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, moduleContract, 0))
	setContractMetadata(t, accountKeeper, ctx, moduleContract)
	require.ErrorContains(t, accountKeeper.registry.RebuildWasmAliases(ctx), "protected module")
	require.Equal(t, expected, snapshotAliases(t, accountKeeper, ctx), "module-key failure must preserve prior registry")
}

func TestRebuildWasmAliasesPreservesStateOnStoreFailure(t *testing.T) {
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	storeKey := storetypes.NewKVStoreKey("wasm")
	ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{authtypes.StoreKey: authKey, "wasm": storeKey}, nil, nil)
	baseService := runtime.NewKVStoreService(storeKey)
	deleteCalls := 0
	service := failingDeleteStoreService{
		KVStoreService: baseService,
		deleteCalls:    &deleteCalls,
		failDeleteAt:   2,
	}
	accountKeeper := newTestAccountKeeperWithService(t, runtime.NewKVStoreService(authKey), service)

	for i := byte(0); i < 2; i++ {
		legacyAlias := sdk.AccAddress(bytes.Repeat([]byte{0xc7 + i}, 20))
		legacyTarget := sdk.AccAddress(append(bytes.Repeat([]byte{0xd7 + i}, 12), legacyAlias...))
		accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, legacyTarget, 0))
		require.NoError(t, accountKeeper.registry.aliases.Set(ctx, legacyAlias, legacyTarget))
	}
	expected := snapshotAliases(t, accountKeeper, ctx)

	newAlias := sdk.AccAddress(bytes.Repeat([]byte{0xd7}, 20))
	newContract := sdk.AccAddress(append(bytes.Repeat([]byte{0xd8}, 12), newAlias...))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, newContract, 0))
	setContractMetadata(t, accountKeeper, ctx, newContract)
	err := accountKeeper.registry.RebuildWasmAliases(ctx)
	require.ErrorContains(t, err, "injected alias delete failure")
	require.Equal(t, 2, deleteCalls, "one cached delete must succeed before the injected failure")
	require.Equal(t, expected, snapshotAliases(t, accountKeeper, ctx))
}

func newTestAccountKeeper(t *testing.T) (testAccountKeeper, sdk.Context) {
	t.Helper()
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	storeKey := storetypes.NewKVStoreKey("wasm")
	ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{authtypes.StoreKey: authKey, "wasm": storeKey}, nil, nil)
	return newTestAccountKeeperWithService(t, runtime.NewKVStoreService(authKey), runtime.NewKVStoreService(storeKey)), ctx
}

type testAccountKeeper struct {
	authkeeper.AccountKeeper
	registry    Keeper
	wasmService corestore.KVStoreService
	cdc         codec.Codec
}

func newTestAccountKeeperWithService(t *testing.T, authService, wasmService corestore.KVStoreService) testAccountKeeper {
	t.Helper()
	interfaceRegistry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	accountKeeper := authkeeper.NewAccountKeeper(
		cdc,
		authService,
		authtypes.ProtoBaseAccount,
		map[string][]string{
			govtypes.ModuleName:  {authtypes.Burner},
			minttypes.ModuleName: {authtypes.Minter},
		},
		authcodec.NewBech32Codec(sdk.Bech32MainPrefix),
		sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)
	base := upstream.NewKeeper(cdc, wasmService, accountKeeper, metadataOnlyBank{}, nil, nil, nil, nil, nil, nil, nil,
		t.TempDir(), wasmtypes.NodeConfig{}, wasmtypes.VMConfig{}, nil, authtypes.NewModuleAddress(govtypes.ModuleName).String(), upstream.WithWasmEngine(metadataOnlyEngine{}))
	return testAccountKeeper{AccountKeeper: accountKeeper, wasmService: wasmService, cdc: cdc, registry: NewKeeper(base, wasmService, accountKeeper, map[string][]string{
		govtypes.ModuleName:  {authtypes.Burner},
		minttypes.ModuleName: {authtypes.Minter},
	})}
}

func newBaseAccount(t *testing.T, accountKeeper testAccountKeeper, ctx sdk.Context, address sdk.AccAddress, sequence uint64) sdk.AccountI {
	t.Helper()
	account := accountKeeper.NewAccountWithAddress(ctx, address)
	require.NoError(t, account.SetSequence(sequence))
	return account
}

func snapshotAliases(t *testing.T, accountKeeper testAccountKeeper, ctx sdk.Context) map[string]string {
	t.Helper()
	result := map[string]string{}
	err := accountKeeper.registry.aliases.Walk(ctx, nil, func(key, value sdk.AccAddress) (bool, error) {
		result[string(key)] = string(value)
		return false, nil
	})
	require.NoError(t, err)
	return result
}

type failingDeleteStoreService struct {
	corestore.KVStoreService
	deleteCalls  *int
	failDeleteAt int
}

func (s failingDeleteStoreService) OpenKVStore(ctx context.Context) corestore.KVStore {
	return failingDeleteStore{
		KVStore:      s.KVStoreService.OpenKVStore(ctx),
		deleteCalls:  s.deleteCalls,
		failDeleteAt: s.failDeleteAt,
	}
}

type failingDeleteStore struct {
	corestore.KVStore
	deleteCalls  *int
	failDeleteAt int
}

func (s failingDeleteStore) Delete(key []byte) error {
	if bytes.HasPrefix(key, types.AliasStoreKeyPrefix.Bytes()) {
		*s.deleteCalls = *s.deleteCalls + 1
		if *s.deleteCalls == s.failDeleteAt {
			return errors.New("injected alias delete failure")
		}
	}
	return s.KVStore.Delete(key)
}

// metadataOnlyEngine avoids constructing a VM for metadata-only keeper tests.
type metadataOnlyEngine struct{ wasmtypes.WasmEngine }

type metadataOnlyBank struct{ wasmtypes.BankKeeper }

func setContractMetadata(t *testing.T, keeper testAccountKeeper, ctx sdk.Context, addresses ...sdk.AccAddress) {
	t.Helper()
	store := keeper.wasmService.OpenKVStore(ctx)
	var oldAddresses []sdk.AccAddress
	keeper.registry.IterateContractInfo(ctx, func(addr sdk.AccAddress, _ wasmtypes.ContractInfo) bool {
		oldAddresses = append(oldAddresses, bytes.Clone(addr))
		return false
	})
	for _, addr := range oldAddresses {
		require.NoError(t, store.Delete(wasmtypes.GetContractAddressKey(addr)))
	}
	for _, addr := range addresses {
		require.NoError(t, store.Set(wasmtypes.GetContractAddressKey(addr), keeper.cdc.MustMarshal(&wasmtypes.ContractInfo{})))
	}
}

func TestRebuildWasmAliasesUsesContractMetadata(t *testing.T) {
	keeper, ctx := newTestAccountKeeper(t)
	contract := sdk.AccAddress(bytes.Repeat([]byte{0x61}, 32))
	generic := sdk.AccAddress(bytes.Repeat([]byte{0x62}, 32))
	for _, addr := range []sdk.AccAddress{contract, generic} {
		keeper.SetAccount(ctx, newBaseAccount(t, keeper, ctx, addr, 0))
	}
	setContractMetadata(t, keeper, ctx, contract)
	require.NoError(t, keeper.registry.RebuildWasmAliases(ctx))
	require.Equal(t, map[string]string{string(contract[12:]): string(contract)}, snapshotAliases(t, keeper, ctx))
}

func TestRebuildWasmAliasesPreservesStateOnInvalidMetadata(t *testing.T) {
	for _, tc := range []struct {
		name          string
		address       sdk.AccAddress
		accountExists bool
		wantError     string
	}{
		{"invalid address length", sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20)), true, "must be 32 bytes"},
		{"missing account", sdk.AccAddress(bytes.Repeat([]byte{0x72}, 32)), false, "account does not exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := newTestAccountKeeper(t)
			existing := sdk.AccAddress(bytes.Repeat([]byte{0x73}, 32))
			keeper.SetAccount(ctx, newBaseAccount(t, keeper, ctx, existing, 0))
			require.NoError(t, keeper.registry.RegisterWasmAlias(ctx, existing[12:], existing))
			expected := snapshotAliases(t, keeper, ctx)
			if tc.accountExists {
				keeper.SetAccount(ctx, newBaseAccount(t, keeper, ctx, tc.address, 0))
			}
			setContractMetadata(t, keeper, ctx, tc.address)
			require.ErrorContains(t, keeper.registry.RebuildWasmAliases(ctx), tc.wantError)
			require.Equal(t, expected, snapshotAliases(t, keeper, ctx))
		})
	}
}

func TestResolveContractAddress(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		alias, exact, account, metadata bool
		wantError                       string
	}{
		{name: "exact contract", account: true, metadata: true},
		{name: "missing account and metadata", wantError: "no such contract"},
		{name: "missing exact account", metadata: true, wantError: "no such contract"},
		{name: "missing exact metadata", account: true, wantError: "no such contract"},
		{name: "alias contract", alias: true, account: true, metadata: true},
		{name: "alias wins over exact account", alias: true, exact: true, account: true, metadata: true},
		{name: "missing alias metadata", alias: true, account: true, wantError: "no such contract"},
		{name: "missing alias account", alias: true, metadata: true, wantError: "account does not exist"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			keeper, ctx := newTestAccountKeeper(t)
			address := sdk.AccAddress(bytes.Repeat([]byte{0xe1}, 20))
			target := address
			if tc.alias {
				target = sdk.AccAddress(append(bytes.Repeat([]byte{0xe2}, 12), address...))
				require.NoError(t, keeper.registry.aliases.Set(ctx, address, target))
			}
			if tc.account {
				keeper.SetAccount(ctx, newBaseAccount(t, keeper, ctx, target, 0))
			}
			if tc.exact {
				keeper.SetAccount(ctx, newBaseAccount(t, keeper, ctx, address, 0))
			}
			if tc.metadata {
				setContractMetadata(t, keeper, ctx, target)
			}
			resolved, err := keeper.registry.ResolveContractAddress(ctx, address)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.Nil(t, resolved)
				if tc.wantError == "no such contract" {
					require.EqualError(t, err, wasmtypes.ErrNoSuchContractFn(address.String()).Error())
				}
				return
			}
			require.NoError(t, err)
			require.Equal(t, target, resolved)
		})
	}
}
