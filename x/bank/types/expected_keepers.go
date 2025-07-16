package types

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

type EvmKeeper interface {
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer *tracing.Hooks, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	GetNonce(ctx sdk.Context, addr common.Address) uint64
}

type WasmMsgServer interface {
	ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
}

type WasmKeeper interface {
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
}
