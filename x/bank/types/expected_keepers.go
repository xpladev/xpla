package types

import (
	"context"
	"math/big"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/evm/x/vm/statedb"
)

type EvmKeeper interface {
	statedb.Keeper
	ApplyMessage(ctx sdk.Context, stateDB *statedb.StateDB, msg core.Message, tracer *tracing.Hooks, commit bool, callFromPrecompile bool, internal bool) (*evmtypes.MsgEthereumTxResponse, error)
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	GetNonce(ctx sdk.Context, addr common.Address) uint64
	CallEVM(
		ctx sdk.Context,
		stateDB *statedb.StateDB,
		abi abi.ABI,
		from, contract common.Address,
		commit bool,
		callFromPrecompile bool,
		gasCap *big.Int,
		method string,
		args ...interface{},
	) (*evmtypes.MsgEthereumTxResponse, error)
}

type WasmMsgServer interface {
	ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
}

type WasmKeeper interface {
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
}
