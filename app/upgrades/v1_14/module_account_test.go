package v1_14

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	"github.com/stretchr/testify/require"

	xplaauthkeeper "github.com/xpladev/xpla/x/auth/keeper"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestEnsureDynamicDeflationPoolAccount(t *testing.T) {
	ctx, accountKeeper := newModuleAccountTestKeeper(t, map[string][]string{
		dynamicdeflationtypes.PoolName: {authtypes.Burner},
	})
	moduleAddress := accountKeeper.GetModuleAddress(dynamicdeflationtypes.PoolName)

	t.Run("creates missing account", func(t *testing.T) {
		cacheCtx, _ := ctx.CacheContext()
		require.Nil(t, accountKeeper.GetAccount(cacheCtx, moduleAddress))

		require.NoError(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper))

		moduleAccount := requireDynamicDeflationPoolAccount(t, cacheCtx, accountKeeper)
		require.Zero(t, moduleAccount.GetSequence())
		require.NoError(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper))
		require.Equal(t, moduleAccount, requireDynamicDeflationPoolAccount(t, cacheCtx, accountKeeper))
		nextAccountNumber, err := accountKeeper.AccountNumber.Peek(cacheCtx)
		require.NoError(t, err)
		require.Equal(t, moduleAccount.GetAccountNumber()+1, nextAccountNumber)
	})

	t.Run("converts base account without changing identity", func(t *testing.T) {
		cacheCtx, _ := ctx.CacheContext()
		baseAccount := authtypes.NewBaseAccount(moduleAddress, nil, 42, 7)
		accountKeeper.SetAccount(cacheCtx, baseAccount)

		require.NoError(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper))

		moduleAccount := requireDynamicDeflationPoolAccount(t, cacheCtx, accountKeeper)
		require.Equal(t, uint64(42), moduleAccount.GetAccountNumber())
		require.Equal(t, uint64(7), moduleAccount.GetSequence())
	})

	t.Run("normalizes existing module name and permissions", func(t *testing.T) {
		cacheCtx, _ := ctx.CacheContext()
		baseAccount := authtypes.NewBaseAccount(moduleAddress, nil, 43, 8)
		accountKeeper.SetModuleAccount(
			cacheCtx,
			authtypes.NewModuleAccount(baseAccount, "wrong_module", authtypes.Minter),
		)

		require.NoError(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper))

		moduleAccount := requireDynamicDeflationPoolAccount(t, cacheCtx, accountKeeper)
		require.Equal(t, uint64(43), moduleAccount.GetAccountNumber())
		require.Equal(t, uint64(8), moduleAccount.GetSequence())
	})

	t.Run("rejects unsupported account without changing it", func(t *testing.T) {
		cacheCtx, _ := ctx.CacheContext()
		account, err := vestingtypes.NewBaseVestingAccount(
			authtypes.NewBaseAccount(moduleAddress, nil, 44, 9), sdk.NewCoins(sdk.NewInt64Coin("axpla", 1)), 100,
		)
		require.NoError(t, err)
		accountKeeper.SetAccount(cacheCtx, account)
		storedAccount := accountKeeper.GetAccount(cacheCtx, moduleAddress)

		require.ErrorContains(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper), "unsupported account type")
		require.Equal(t, storedAccount, accountKeeper.GetAccount(cacheCtx, moduleAddress))
	})

	t.Run("rejects mismatched public key without changing account", func(t *testing.T) {
		cacheCtx, _ := ctx.CacheContext()
		pubKey := secp256k1.GenPrivKeyFromSecret([]byte("pool account test")).PubKey()
		baseAccount := authtypes.NewBaseAccount(moduleAddress, pubKey, 45, 10)
		accountKeeper.SetAccount(cacheCtx, baseAccount)

		require.ErrorContains(t, ensureDynamicDeflationPoolAccount(cacheCtx, accountKeeper), "account address and pubkey address do not match")
		require.Equal(t, baseAccount, accountKeeper.GetAccount(cacheCtx, moduleAddress))
	})
}

func TestEnsureDynamicDeflationPoolAccountRejectsInvalidConfiguration(t *testing.T) {
	t.Run("mismatched module address", func(t *testing.T) {
		ctx, accountKeeper := newModuleAccountTestKeeper(t, map[string][]string{
			dynamicdeflationtypes.PoolName: {authtypes.Burner},
		})
		accountKeeper.GetModulePermissions()[dynamicdeflationtypes.PoolName] =
			authtypes.NewPermissionsForAddress("wrong_module", []string{authtypes.Burner})

		require.Error(t, ensureDynamicDeflationPoolAccount(ctx, accountKeeper))
	})

	for _, tc := range []struct {
		name        string
		permissions map[string][]string
		err         string
	}{
		{"missing module", nil, "module address is not configured"},
		{"missing burner", map[string][]string{dynamicdeflationtypes.PoolName: {}}, "module permissions"},
		{"extra permission", map[string][]string{dynamicdeflationtypes.PoolName: {authtypes.Burner, authtypes.Minter}}, "module permissions"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, accountKeeper := newModuleAccountTestKeeper(t, tc.permissions)

			require.ErrorContains(t, ensureDynamicDeflationPoolAccount(ctx, accountKeeper), tc.err)
			require.Nil(t, accountKeeper.GetAccount(ctx, authtypes.NewModuleAddress(dynamicdeflationtypes.PoolName)))
		})
	}
}

func newModuleAccountTestKeeper(t *testing.T, permissions map[string][]string) (sdk.Context, xplaauthkeeper.AccountKeeper) {
	t.Helper()
	authKey := storetypes.NewKVStoreKey(authtypes.StoreKey)
	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{authtypes.StoreKey: authKey},
		nil,
		nil,
	)
	registry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	cryptocodec.RegisterInterfaces(registry)
	authtypes.RegisterInterfaces(registry)
	vestingtypes.RegisterInterfaces(registry)
	accountKeeper := xplaauthkeeper.NewAccountKeeper(authkeeper.NewAccountKeeper(
		codec.NewProtoCodec(registry),
		runtime.NewKVStoreService(authKey),
		authtypes.ProtoBaseAccount,
		permissions,
		authcodec.NewBech32Codec(sdk.Bech32MainPrefix),
		sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress("gov").String(),
	))

	return ctx, accountKeeper
}

func requireDynamicDeflationPoolAccount(
	t *testing.T,
	ctx sdk.Context,
	accountKeeper xplaauthkeeper.AccountKeeper,
) sdk.ModuleAccountI {
	t.Helper()
	moduleAddress := accountKeeper.GetModuleAddress(dynamicdeflationtypes.PoolName)
	account := accountKeeper.GetAccount(ctx, moduleAddress)
	moduleAccount, ok := account.(sdk.ModuleAccountI)
	require.True(t, ok)
	require.Equal(t, dynamicdeflationtypes.PoolName, moduleAccount.GetName())
	require.Equal(t, moduleAddress, moduleAccount.GetAddress())
	require.Equal(t, []string{authtypes.Burner}, moduleAccount.GetPermissions())
	return moduleAccount
}
