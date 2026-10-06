package wasm

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type WasmMsgServer interface {
	InstantiateContract(ctx context.Context, msg *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error)
	InstantiateContract2(ctx context.Context, msg *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error)
	ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
	MigrateContract(ctx context.Context, msg *wasmtypes.MsgMigrateContract) (*wasmtypes.MsgMigrateContractResponse, error)
}

type WasmKeeper interface {
	ResolveContractAddress(ctx context.Context, evmAddr sdk.AccAddress) (sdk.AccAddress, error)
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
}
