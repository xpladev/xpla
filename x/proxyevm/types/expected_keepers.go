package types

import (
	context "context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

// AccountKeeper defines the expected interface needed to retrieve account info.
type AccountKeeper interface {
	GetSequence(sdk.Context, sdk.AccAddress) (uint64, error)
}

// EVMKeeper defines the expected EVM keeper interface used on proxy evm
type EVMKeeper interface {
	GetParams(ctx sdk.Context) (params evmtypes.Params)
	GetTxIndexTransient(ctx sdk.Context) uint64
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
}
