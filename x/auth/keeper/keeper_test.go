package keeper

import (
	"bytes"
	"testing"

	storetypes "cosmossdk.io/store/types"
	upstreamwasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	wasmkeeper "github.com/xpladev/xpla/x/wasm/keeper"
)

func TestAccountKeeperResolveAccountAddress(t *testing.T) {
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	target := sdk.AccAddress(append(bytes.Repeat([]byte{0x32}, 12), alias...))
	tests := []struct {
		name          string
		requested     sdk.AccAddress
		expected      sdk.AccAddress
		setup         func(sdk.Context, AccountKeeper, sdk.AccountI)
		expectedError string
	}{
		{
			name:      "alias",
			requested: alias,
			expected:  target,
		},
		{
			name:      "exact over alias",
			requested: alias,
			expected:  alias,
			setup: func(ctx sdk.Context, accounts AccountKeeper, account sdk.AccountI) {
				accounts.SetAccount(ctx, accounts.NewAccountWithAddress(ctx, alias))
				// A stale alias must not affect a valid exact account.
				accounts.RemoveAccount(ctx, account)
			},
		},
		{
			name:      "canonical",
			requested: target,
			expected:  target,
		},
		{
			name:      "missing",
			requested: sdk.AccAddress(bytes.Repeat([]byte{0x45}, 20)),
		},
		{
			name:      "missing long address",
			requested: sdk.AccAddress(bytes.Repeat([]byte{0x45}, 32)),
		},
		{
			name:          "stale target",
			requested:     alias,
			expectedError: "wasm contract account does not exist",
			setup: func(ctx sdk.Context, accounts AccountKeeper, account sdk.AccountI) {
				accounts.RemoveAccount(ctx, account)
			},
		},
		{
			name:          "protected module",
			requested:     authtypes.NewModuleAddress("gov"),
			expectedError: "protected module address",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authKey, wasmKey := storetypes.NewKVStoreKey(authtypes.StoreKey), storetypes.NewKVStoreKey("wasm")
			ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{authtypes.StoreKey: authKey, "wasm": wasmKey}, nil, nil)
			registry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
			authtypes.RegisterInterfaces(registry)
			accounts := NewAccountKeeper(authkeeper.NewAccountKeeper(
				codec.NewProtoCodec(registry), runtime.NewKVStoreService(authKey), authtypes.ProtoBaseAccount,
				map[string][]string{}, authcodec.NewBech32Codec(sdk.Bech32MainPrefix), sdk.Bech32MainPrefix,
				authtypes.NewModuleAddress("gov").String(),
			))
			aliases := wasmkeeper.NewKeeper(upstreamwasmkeeper.Keeper{}, runtime.NewKVStoreService(wasmKey), accounts, map[string][]string{"gov": nil})
			account := accounts.NewAccountWithAddress(ctx, target)
			accounts.SetAccount(ctx, account)
			require.NoError(t, aliases.RegisterWasmAlias(ctx, alias, target))
			require.Nil(t, accounts.GetAccount(ctx, alias), "SDK identity lookup must stay exact")
			require.False(t, accounts.HasAccount(ctx, alias))
			if tt.setup != nil {
				tt.setup(ctx, accounts, account)
			}
			resolved, err := accounts.ResolveAccountAddress(ctx, tt.requested, aliases)
			if tt.expectedError == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.expectedError)
			}
			require.Equal(t, tt.expected, resolved)
		})
	}
}
