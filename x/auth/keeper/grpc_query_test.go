package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	upstreamwasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	xplaauthtypes "github.com/xpladev/xpla/x/auth/types"
	wasmkeeper "github.com/xpladev/xpla/x/wasm/keeper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testWasmKeeper struct {
	xplaauthtypes.WasmKeeper
	err error
}

func (r *testWasmKeeper) ResolveWasmAlias(ctx context.Context, addr sdk.AccAddress) (sdk.AccAddress, bool, error) {
	if r.err != nil {
		return nil, false, r.err
	}
	return r.WasmKeeper.ResolveWasmAlias(ctx, addr)
}

func TestAuthQueryCompatibility(t *testing.T) {
	key := storetypes.NewKVStoreKey(authtypes.StoreKey)
	wasmKey := storetypes.NewKVStoreKey("wasm")
	ctx := testutil.DefaultContextWithKeys(map[string]*storetypes.KVStoreKey{authtypes.StoreKey: key, "wasm": wasmKey}, nil, nil)
	registry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	addressCodec := authcodec.NewBech32Codec(sdk.Bech32MainPrefix)
	keeper := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(key), authtypes.ProtoBaseAccount,
		map[string][]string{}, addressCodec, sdk.Bech32MainPrefix, authtypes.NewModuleAddress("gov").String())
	aliases := wasmkeeper.NewKeeper(upstreamwasmkeeper.Keeper{}, runtime.NewKVStoreService(wasmKey), keeper, nil)
	wasm := &testWasmKeeper{WasmKeeper: aliases}
	server := NewQueryServer(NewAccountKeeper(keeper), wasm)
	upstream := authkeeper.NewQueryServer(keeper)
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x34}, 20))
	contract := sdk.AccAddress(append(bytes.Repeat([]byte{0x56}, 12), alias...))
	contractAccount := keeper.NewAccountWithAddress(ctx, contract)
	require.NoError(t, contractAccount.SetSequence(19))
	keeper.SetAccount(ctx, contractAccount)
	require.NoError(t, aliases.RegisterWasmAlias(ctx, alias, contract))
	aliasString, err := addressCodec.BytesToString(alias)
	require.NoError(t, err)
	contractString, err := addressCodec.BytesToString(contract)
	require.NoError(t, err)

	for _, exact := range []bool{false, true} {
		if exact {
			account := keeper.NewAccountWithAddress(ctx, alias)
			require.NoError(t, account.SetSequence(27))
			keeper.SetAccount(ctx, account)
		}
		canonical := contractString
		if exact {
			canonical = aliasString
		}
		request := &authtypes.QueryAccountRequest{Address: aliasString}
		actual, err := server.Account(ctx, request)
		require.NoError(t, err)
		expected, err := upstream.Account(ctx, &authtypes.QueryAccountRequest{Address: canonical})
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		require.Equal(t, aliasString, request.Address)
		infoRequest := &authtypes.QueryAccountInfoRequest{Address: aliasString}
		info, err := server.AccountInfo(ctx, infoRequest)
		require.NoError(t, err)
		expectedInfo, err := upstream.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: canonical})
		require.NoError(t, err)
		expectedInfo.Info.Address = aliasString
		require.Equal(t, expectedInfo, info)
		require.Equal(t, aliasString, infoRequest.Address)
	}

	missing, err := addressCodec.BytesToString(bytes.Repeat([]byte{0x78}, 20))
	require.NoError(t, err)
	for _, requested := range []string{"", "invalid", missing} {
		_, actualErr := server.Account(ctx, &authtypes.QueryAccountRequest{Address: requested})
		_, expectedErr := upstream.Account(ctx, &authtypes.QueryAccountRequest{Address: requested})
		require.EqualError(t, actualErr, expectedErr.Error())
		_, actualErr = server.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: requested})
		_, expectedErr = upstream.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: requested})
		require.EqualError(t, actualErr, expectedErr.Error())
	}
	_, err = server.Account(ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	_, err = server.AccountInfo(ctx, nil)
	require.Equal(t, codes.InvalidArgument, status.Code(err))
	// Exact accounts do not consult Wasm. Remove the exact account to exercise
	// propagation of an alias lookup failure through the RPC boundary.
	keeper.RemoveAccount(ctx, keeper.GetAccount(ctx, alias))
	wasm.err = errors.New("registry unavailable")
	_, err = server.Account(ctx, &authtypes.QueryAccountRequest{Address: aliasString})
	require.EqualError(t, err, "rpc error: code = Internal desc = resolve account: registry unavailable")
	_, err = server.AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: aliasString})
	require.Equal(t, codes.Internal, status.Code(err))
}
