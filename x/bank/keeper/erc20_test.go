package keeper

import (
	"context"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/stretchr/testify/require"

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
	callResponse       *evmtypes.MsgEthereumTxResponse
	callErr            error
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
	_ sdk.Context,
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
	return e.callResponse, e.callErr
}

func TestExecuteTransferBoundsPrecompileCallByRemainingGasAndConsumesMaxUsedGas(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
		gasUsed     = uint64(1_000)
		maxUsedGas  = uint64(1_234)
	)

	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit))
	ctx.GasMeter().ConsumeGas(consumedGas, "test setup")

	from := common.HexToAddress("0x1000")
	contract := common.HexToAddress("0x2000")
	to := common.HexToAddress("0x3000")
	amount := big.NewInt(7)
	returnData, err := ABI.Methods[banktypes.GetErc20Method(banktypes.Transfer)].Outputs.Pack(true)
	require.NoError(t, err)

	executor := &recordingERC20EVMExecutor{
		nonce: 9,
		applyResponse: &evmtypes.MsgEthereumTxResponse{
			GasUsed:    gasUsed,
			MaxUsedGas: maxUsedGas,
			Ret:        returnData,
		},
	}
	stateDB := &statedb.StateDB{}
	ctx = banktypes.WithEVMStateDB(ctx, stateDB)
	keeper := Erc20Keeper{ek: executor}

	err = keeper.ExecuteTransfer(ctx, contract, sdk.AccAddress(from.Bytes()), sdk.AccAddress(to.Bytes()), amount)
	require.NoError(t, err)
	require.Equal(t, 1, executor.applyCalls)
	require.Zero(t, executor.callCalls)
	require.Same(t, stateDB, executor.appliedStateDB)
	require.Equal(t, gasLimit-consumedGas, executor.appliedMessage.GasLimit)
	require.Equal(t, from, executor.appliedMessage.From)
	require.Equal(t, contract, *executor.appliedMessage.To)
	require.Equal(t, uint64(9), executor.appliedMessage.Nonce)
	require.False(t, executor.applyCommit)
	require.True(t, executor.applyCallFromPrecompile)
	require.True(t, executor.applyInternal)
	require.Equal(t, consumedGas+maxUsedGas, ctx.GasMeter().GasConsumed())
}

func TestExecuteTransferKeepsNonPrecompileCallEVMPath(t *testing.T) {
	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(100_000))
	from := common.HexToAddress("0x1000")
	contract := common.HexToAddress("0x2000")
	to := common.HexToAddress("0x3000")
	amount := big.NewInt(7)
	returnData, err := ABI.Methods[banktypes.GetErc20Method(banktypes.Transfer)].Outputs.Pack(true)
	require.NoError(t, err)
	executor := &recordingERC20EVMExecutor{
		callResponse: &evmtypes.MsgEthereumTxResponse{Ret: returnData},
	}
	keeper := Erc20Keeper{ek: executor}

	err = keeper.ExecuteTransfer(
		ctx,
		contract,
		sdk.AccAddress(from.Bytes()),
		sdk.AccAddress(to.Bytes()),
		amount,
	)
	require.NoError(t, err)
	require.Zero(t, executor.applyCalls)
	require.Equal(t, 1, executor.callCalls)
	require.NotNil(t, executor.callStateDB)
	require.Equal(t, from, executor.callFrom)
	require.Equal(t, contract, executor.callContract)
	require.True(t, executor.callCommit)
	require.False(t, executor.callFromPrecompile)
	require.Nil(t, executor.callGasCap)
	require.Equal(t, banktypes.GetErc20Method(banktypes.Transfer), executor.callMethod)
	require.Equal(t, []interface{}{to, amount}, executor.callArgs)
}

func TestExecuteTransferConsumesAllGasOnFailedPrecompileCall(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
	)

	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit))
	ctx.GasMeter().ConsumeGas(consumedGas, "test setup")

	response := &evmtypes.MsgEthereumTxResponse{VmError: vm.ErrOutOfGas.Error()}
	executor := &recordingERC20EVMExecutor{applyResponse: response}
	stateDB := &statedb.StateDB{}
	ctx = banktypes.WithEVMStateDB(ctx, stateDB)
	keeper := Erc20Keeper{ek: executor}
	from := common.HexToAddress("0x1000")
	contract := common.HexToAddress("0x2000")

	err := keeper.ExecuteTransfer(
		ctx,
		contract,
		sdk.AccAddress(from.Bytes()),
		sdk.AccAddress(common.HexToAddress("0x3000").Bytes()),
		big.NewInt(7),
	)
	require.ErrorContains(t, err, vm.ErrOutOfGas.Error())
	require.Equal(t, 1, executor.applyCalls)
	require.Zero(t, executor.callCalls)
	require.Same(t, stateDB, executor.appliedStateDB)
	require.Equal(t, gasLimit-consumedGas, executor.appliedMessage.GasLimit)
	require.Equal(t, from, executor.appliedMessage.From)
	require.Equal(t, contract, *executor.appliedMessage.To)
	require.Equal(t, gasLimit, ctx.GasMeter().GasConsumed())
	require.True(t, ctx.GasMeter().IsOutOfGas())
}

func TestExecuteTransferConsumesAllGasOnPrecompileApplyError(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
	)

	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit))
	ctx.GasMeter().ConsumeGas(consumedGas, "test setup")

	executor := &recordingERC20EVMExecutor{applyErr: core.ErrIntrinsicGas}
	stateDB := &statedb.StateDB{}
	ctx = banktypes.WithEVMStateDB(ctx, stateDB)
	keeper := Erc20Keeper{ek: executor}

	err := keeper.ExecuteTransfer(
		ctx,
		common.HexToAddress("0x2000"),
		sdk.AccAddress(common.HexToAddress("0x1000").Bytes()),
		sdk.AccAddress(common.HexToAddress("0x3000").Bytes()),
		big.NewInt(7),
	)
	require.ErrorContains(t, err, core.ErrIntrinsicGas.Error())
	require.Equal(t, 1, executor.applyCalls)
	require.Zero(t, executor.callCalls)
	require.Same(t, stateDB, executor.appliedStateDB)
	require.Equal(t, gasLimit-consumedGas, executor.appliedMessage.GasLimit)
	require.Equal(t, gasLimit, ctx.GasMeter().GasConsumed())
	require.True(t, ctx.GasMeter().IsOutOfGas())
}
