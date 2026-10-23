package keeper

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"cosmossdk.io/collections"
	ccodec "cosmossdk.io/collections/codec"
	corestore "cosmossdk.io/core/store"

	upstream "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/xpladev/xpla/x/wasm/types"
)

// Keeper extends the Wasm keeper with compatibility aliases in the Wasm store.
type Keeper struct {
	upstream.Keeper
	aliases                  collections.Map[sdk.AccAddress, sdk.AccAddress]
	accountKeeper            types.AccountKeeper
	protectedModuleAddresses map[string]struct{}
}

func NewKeeper(base upstream.Keeper, storeService corestore.KVStoreService, accountKeeper types.AccountKeeper, maccPerms map[string][]string) Keeper {
	sb := collections.NewSchemaBuilder(storeService)
	protected := make(map[string]struct{}, len(maccPerms))
	for moduleName := range maccPerms {
		protected[string(authtypes.NewModuleAddress(moduleName))] = struct{}{}
	}
	return Keeper{
		Keeper:                   base,
		aliases:                  collections.NewMap(sb, types.AliasStoreKeyPrefix, "wasmAliases", sdk.AccAddressKey, ccodec.KeyToValueCodec(sdk.AccAddressKey)),
		accountKeeper:            accountKeeper,
		protectedModuleAddresses: protected,
	}
}

// RegisterWasmAlias records the explicit mapping used by the 20-byte IWasm ABI.
// Generic account storage must never call this method.
func (ak Keeper) RegisterWasmAlias(ctx context.Context, evmAddr, contractAddr sdk.AccAddress) error {
	if err := ak.validateWasmAlias(ctx, evmAddr, contractAddr); err != nil {
		return err
	}

	existing, err := ak.aliases.Get(ctx, evmAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return ak.aliases.Set(ctx, evmAddr, contractAddr)
		}
		return err
	}
	if bytes.Equal(existing, contractAddr) {
		return nil
	}
	return fmt.Errorf("wasm alias %s already resolves to %s", evmAddr, existing)
}

// ResolveWasmAlias resolves only explicitly registered Wasm compatibility aliases.
func (ak Keeper) ResolveWasmAlias(ctx context.Context, evmAddr sdk.AccAddress) (sdk.AccAddress, bool, error) {
	if len(evmAddr) != 20 {
		return nil, false, fmt.Errorf("wasm alias key must be 20 bytes: got %d", len(evmAddr))
	}
	if ak.isProtectedModuleAddress(evmAddr) {
		return nil, false, fmt.Errorf("wasm alias key is a protected module address: %s", evmAddr)
	}

	contractAddr, err := ak.aliases.Get(ctx, evmAddr)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, false, nil
		}
		return nil, false, err
	}
	if err := ak.validateWasmAlias(ctx, evmAddr, contractAddr); err != nil {
		return nil, false, err
	}
	return contractAddr, true, nil
}

// ResolveContractAddress selects the registered Wasm alias before an exact
// address and verifies that the selected account has contract metadata.
func (ak Keeper) ResolveContractAddress(ctx context.Context, evmAddr sdk.AccAddress) (sdk.AccAddress, error) {
	contractAddr, found, err := ak.ResolveWasmAlias(ctx, evmAddr)
	if err != nil {
		return nil, err
	}
	if !found {
		contractAddr = evmAddr
	}
	if !ak.accountKeeper.HasAccount(ctx, contractAddr) || ak.GetContractInfo(ctx, contractAddr) == nil {
		return nil, wasmtypes.ErrNoSuchContractFn(evmAddr.String())
	}
	return contractAddr, nil
}

// RebuildWasmAliases atomically replaces the compatibility registry from
// authoritative Wasm contract addresses.
func (ak Keeper) RebuildWasmAliases(ctx context.Context) error {
	var candidates []sdk.AccAddress
	ak.IterateContractInfo(ctx, func(addr sdk.AccAddress, _ wasmtypes.ContractInfo) bool {
		candidates = append(candidates, bytes.Clone(addr))
		return false
	})
	sort.Slice(candidates, func(i, j int) bool { return bytes.Compare(candidates[i], candidates[j]) < 0 })

	byAlias := make(map[string]sdk.AccAddress, len(candidates))
	for _, contractAddr := range candidates {
		if len(contractAddr) != 32 {
			return fmt.Errorf("authoritative wasm contract address must be 32 bytes: got %d", len(contractAddr))
		}
		evmAddr := sdk.AccAddress(bytes.Clone(contractAddr[len(contractAddr)-20:]))
		if err := ak.validateWasmAlias(ctx, evmAddr, contractAddr); err != nil {
			return err
		}
		key := string(evmAddr)
		if existing, ok := byAlias[key]; ok && !bytes.Equal(existing, contractAddr) {
			return fmt.Errorf("authoritative wasm contracts %s and %s share alias %s", existing, contractAddr, evmAddr)
		}
		byAlias[key] = contractAddr
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	cacheCtx, write := sdkCtx.CacheContext()
	if err := ak.aliases.Clear(cacheCtx, nil); err != nil {
		return err
	}
	for _, contractAddr := range candidates {
		evmAddr := sdk.AccAddress(bytes.Clone(contractAddr[len(contractAddr)-20:]))
		if err := ak.RegisterWasmAlias(cacheCtx, evmAddr, contractAddr); err != nil {
			return err
		}
	}
	write()
	return nil
}

func (ak Keeper) validateWasmAlias(ctx context.Context, evmAddr, contractAddr sdk.AccAddress) error {
	if len(evmAddr) != 20 {
		return fmt.Errorf("wasm alias key must be 20 bytes: got %d", len(evmAddr))
	}
	if len(contractAddr) != 32 {
		return fmt.Errorf("wasm contract address must be 32 bytes: got %d", len(contractAddr))
	}
	if !bytes.Equal(evmAddr, contractAddr[len(contractAddr)-20:]) {
		return fmt.Errorf("wasm alias %s does not match contract suffix %s", evmAddr, contractAddr)
	}
	if ak.isProtectedModuleAddress(evmAddr) {
		return fmt.Errorf("wasm alias key is a protected module address: %s", evmAddr)
	}
	if ak.accountKeeper.GetAccount(ctx, contractAddr) == nil {
		return fmt.Errorf("wasm contract account does not exist: %s", contractAddr)
	}
	return nil
}

func (ak Keeper) isProtectedModuleAddress(addr sdk.AccAddress) bool {
	_, ok := ak.protectedModuleAddresses[string(addr)]
	return ok
}
