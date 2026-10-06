package keeper

import (
	"bytes"
	"context"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestMsgServerRegistersEveryContractCreationRoute(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	contract := sdk.AccAddress(bytes.Repeat([]byte{0x42}, 32))
	accountKeeper.SetAccount(ctx, newBaseAccount(t, accountKeeper, ctx, contract, 0))
	delegate := creationMsgServer{address: contract.String()}
	server := msgServer{MsgServer: delegate, Keeper: &accountKeeper.registry}
	checkAlias := func() {
		t.Helper()
		resolved, found, err := accountKeeper.registry.ResolveWasmAlias(ctx, contract[12:])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, contract, resolved)
		// Each creation route must register the alias independently.
		require.NoError(t, accountKeeper.registry.aliases.Remove(ctx, contract[12:]))
	}

	_, err := server.InstantiateContract(ctx, &wasmtypes.MsgInstantiateContract{})
	require.NoError(t, err)
	checkAlias()
	_, err = server.InstantiateContract2(ctx, &wasmtypes.MsgInstantiateContract2{})
	require.NoError(t, err)
	checkAlias()
	_, err = server.StoreAndInstantiateContract(ctx, &wasmtypes.MsgStoreAndInstantiateContract{})
	require.NoError(t, err)
	checkAlias()
}

func TestMsgServerReturnsRegistrationFailure(t *testing.T) {
	accountKeeper, ctx := newTestAccountKeeper(t)
	contract := sdk.AccAddress(bytes.Repeat([]byte{0x24}, 32))
	server := msgServer{
		MsgServer: creationMsgServer{address: contract.String()},
		Keeper:    &accountKeeper.registry,
	}

	// The delegate returns an address without creating its account, so the real
	// alias registry must reject registration on every creation route.
	expected := "register instantiated wasm contract alias: wasm contract account does not exist: " + contract.String()
	_, err := server.InstantiateContract(ctx, &wasmtypes.MsgInstantiateContract{})
	require.EqualError(t, err, expected)
	_, err = server.InstantiateContract2(ctx, &wasmtypes.MsgInstantiateContract2{})
	require.EqualError(t, err, expected)
	_, err = server.StoreAndInstantiateContract(ctx, &wasmtypes.MsgStoreAndInstantiateContract{})
	require.EqualError(t, err, expected)
}

type creationMsgServer struct {
	*wasmtypes.UnimplementedMsgServer
	address string
}

func (s creationMsgServer) InstantiateContract(context.Context, *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	return &wasmtypes.MsgInstantiateContractResponse{Address: s.address}, nil
}

func (s creationMsgServer) InstantiateContract2(context.Context, *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	return &wasmtypes.MsgInstantiateContract2Response{Address: s.address}, nil
}

func (s creationMsgServer) StoreAndInstantiateContract(context.Context, *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	return &wasmtypes.MsgStoreAndInstantiateContractResponse{Address: s.address}, nil
}
