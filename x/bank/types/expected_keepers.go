package types

import (
	"context"
	"math/big"

	sdkmath "cosmossdk.io/math"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"

	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// TestErcKeeper defines the expected interface for the ERC20 module.
type TestErcKeeper interface {
	QueryBalanceOf(ctx sdk.Context, contractAddress common.Address, account sdk.AccAddress) (sdkmath.Int, error)
	QueryTotalSupply(ctx sdk.Context, contractAddress common.Address) (sdkmath.Int, error)
	ExecuteTransfer(ctx sdk.Context, contractAddress common.Address, sender, to sdk.AccAddress, amount *big.Int) error
}

// TestErcKeeper defines the expected interface for the ERC20 module.
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

type GovKeeper interface {
	Proposal(ctx context.Context, req *govv1types.QueryProposalRequest) (*govv1types.QueryProposalResponse, error)
}
