package keeper

import (
	"context"
	"math/big"

	"cosmossdk.io/core/address"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"

	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	banktypes "github.com/xpladev/xpla/x/bank/types"
)

type recordingERC20EVMExecutor struct {
	banktypes.EvmKeeper

	nonce uint64

	applyCalls              int
	appliedStateDB          *statedb.StateDB
	appliedMessage          core.Message
	applyCommit             bool
	applyCallFromPrecompile bool
	applyInternal           bool
	applyResponse           *evmtypes.MsgEthereumTxResponse
	applyErr                error

	callCalls          int
	callStateDB        *statedb.StateDB
	callFrom           common.Address
	callContract       common.Address
	callCommit         bool
	callFromPrecompile bool
	callGasCap         *big.Int
	callMethod         string
	callArgs           []interface{}
	callConsumeGas     uint64
	callResponse       *evmtypes.MsgEthereumTxResponse
	callErr            error
}

type moduleAccountKeeper struct {
	cosmosbanktypes.AccountKeeper

	moduleAccount sdk.ModuleAccountI
}

func (k moduleAccountKeeper) AddressCodec() address.Codec {
	return addresscodec.NewBech32Codec(sdk.Bech32MainPrefix)
}

func (k moduleAccountKeeper) GetModuleAccount(context.Context, string) sdk.ModuleAccountI {
	return k.moduleAccount
}

func (e *recordingERC20EVMExecutor) ApplyMessage(
	_ sdk.Context,
	stateDB *statedb.StateDB,
	msg core.Message,
	_ *tracing.Hooks,
	commit bool,
	callFromPrecompile bool,
	internal bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	e.applyCalls++
	e.appliedStateDB = stateDB
	e.appliedMessage = msg
	e.applyCommit = commit
	e.applyCallFromPrecompile = callFromPrecompile
	e.applyInternal = internal
	return e.applyResponse, e.applyErr
}

func (e *recordingERC20EVMExecutor) GetNonce(_ sdk.Context, _ common.Address) uint64 {
	return e.nonce
}

func (e *recordingERC20EVMExecutor) CallEVM(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	_ abi.ABI,
	from, contract common.Address,
	commit bool,
	callFromPrecompile bool,
	gasCap *big.Int,
	method string,
	args ...interface{},
) (*evmtypes.MsgEthereumTxResponse, error) {
	e.callCalls++
	e.callStateDB = stateDB
	e.callFrom = from
	e.callContract = contract
	e.callCommit = commit
	e.callFromPrecompile = callFromPrecompile
	e.callGasCap = gasCap
	e.callMethod = method
	e.callArgs = args
	if e.callConsumeGas > 0 {
		ctx.GasMeter().ConsumeGas(e.callConsumeGas, "mock EVM call")
	}
	return e.callResponse, e.callErr
}
