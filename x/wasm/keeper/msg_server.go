package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	upstream "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

type msgServer struct {
	wasmtypes.MsgServer
	*Keeper
}

var _ wasmtypes.MsgServer = msgServer{}

// NewMsgServerImpl creates a Wasm message server that registers contract aliases.
func NewMsgServerImpl(k *Keeper) wasmtypes.MsgServer {
	return msgServer{MsgServer: upstream.NewMsgServerImpl(&k.Keeper), Keeper: k}
}

func (m msgServer) InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	response, err := m.MsgServer.InstantiateContract(ctx, msg)
	if err != nil {
		return nil, err
	}
	if err := m.registerResponse(ctx, response.Address); err != nil {
		return nil, err
	}
	return response, nil
}

func (m msgServer) InstantiateContract2(ctx context.Context, msg *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	response, err := m.MsgServer.InstantiateContract2(ctx, msg)
	if err != nil {
		return nil, err
	}
	if err := m.registerResponse(ctx, response.Address); err != nil {
		return nil, err
	}
	return response, nil
}

func (m msgServer) StoreAndInstantiateContract(ctx context.Context, msg *wasmtypes.MsgStoreAndInstantiateContract) (*wasmtypes.MsgStoreAndInstantiateContractResponse, error) {
	response, err := m.MsgServer.StoreAndInstantiateContract(ctx, msg)
	if err != nil {
		return nil, err
	}
	if err := m.registerResponse(ctx, response.Address); err != nil {
		return nil, err
	}
	return response, nil
}

func (m msgServer) registerResponse(ctx context.Context, address string) error {
	contractAddr, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		return fmt.Errorf("decode instantiated wasm contract address: %w", err)
	}
	if len(contractAddr) != 32 {
		return fmt.Errorf("instantiated wasm contract address must be 32 bytes: got %d", len(contractAddr))
	}
	evmAddr := sdk.AccAddress(contractAddr[len(contractAddr)-20:])
	if err := m.RegisterWasmAlias(ctx, evmAddr, contractAddr); err != nil {
		return fmt.Errorf("register instantiated wasm contract alias: %w", err)
	}
	return nil
}
