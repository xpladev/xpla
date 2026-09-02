package keeper

import (
	"context"
	"errors"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	cosmosbanktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	banktypes "github.com/xpladev/xpla/x/bank/types"
)

type queryAccountKeeper struct {
	cosmosbanktypes.AccountKeeper
}

func TestErc20ViewKeeperRejectsInvalidContractAddressWithoutCallingEVM(t *testing.T) {
	executor := &recordingERC20EVMExecutor{}
	viewKeeper := Erc20ViewKeeper{erc20keeper: Erc20Keeper{ek: executor}}

	var coin sdk.Coin
	require.NotPanics(t, func() {
		coin = viewKeeper.GetBalance(context.Background(), sdk.AccAddress{1}, "invalid")
	})

	require.True(t, coin.Amount.IsZero())
	require.Zero(t, executor.callCalls)
}

func TestErc20BaseKeeperRejectsInvalidContractAddressWithoutCallingEVM(t *testing.T) {
	executor := &recordingERC20EVMExecutor{}
	keeper := NewBaseErc20Keeper(nil, executor)

	var coin sdk.Coin
	require.NotPanics(t, func() {
		coin = keeper.GetSupply(context.Background(), "invalid")
	})

	require.True(t, coin.Amount.IsZero())
	require.Zero(t, executor.callCalls)
}

func TestErc20ViewKeeperReturnsZeroOnQueryError(t *testing.T) {
	executor := &recordingERC20EVMExecutor{callErr: errors.New("query failed")}
	viewKeeper := Erc20ViewKeeper{
		erc20keeper: Erc20Keeper{
			ak: moduleAccountKeeper{
				moduleAccount: authtypes.NewEmptyModuleAccount(cosmosbanktypes.ModuleName),
			},
			ek: executor,
		},
	}
	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(100_000))

	var coin sdk.Coin
	require.NotPanics(t, func() {
		coin = viewKeeper.GetBalance(ctx, sdk.AccAddress{1}, "A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546")
	})

	require.True(t, coin.Amount.IsZero())
	require.Equal(t, 1, executor.callCalls)
}

func TestErc20BaseKeeperReturnsZeroOnSupplyQueryError(t *testing.T) {
	executor := &recordingERC20EVMExecutor{callErr: errors.New("query failed")}
	keeper := NewBaseErc20Keeper(
		moduleAccountKeeper{moduleAccount: authtypes.NewEmptyModuleAccount(cosmosbanktypes.ModuleName)},
		executor,
	)
	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(100_000))

	coin := keeper.GetSupply(ctx, "A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546")

	require.True(t, coin.Amount.IsZero())
	require.Equal(t, 1, executor.callCalls)
}

func (queryAccountKeeper) GetModuleAccount(_ context.Context, moduleName string) sdk.ModuleAccountI {
	return authtypes.NewEmptyModuleAccount(moduleName)
}

func TestERC20QueriesPassRemainingGasCap(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
	)

	contract := common.HexToAddress("0x2000")
	account := sdk.AccAddress(common.HexToAddress("0x3000").Bytes())
	totalSupplyReturn, err := ABI.Methods[banktypes.GetErc20Method(banktypes.TotalSupply)].Outputs.Pack(big.NewInt(42))
	require.NoError(t, err)
	balanceReturn, err := ABI.Methods[banktypes.GetErc20Method(banktypes.BalanceOf)].Outputs.Pack(big.NewInt(7))
	require.NoError(t, err)

	testCases := []struct {
		name       string
		method     string
		returnData []byte
		query      func(Erc20Keeper, sdk.Context) error
	}{
		{
			name:       "total supply",
			method:     banktypes.GetErc20Method(banktypes.TotalSupply),
			returnData: totalSupplyReturn,
			query: func(keeper Erc20Keeper, ctx sdk.Context) error {
				_, err := keeper.QueryTotalSupply(ctx, contract)
				return err
			},
		},
		{
			name:       "balance",
			method:     banktypes.GetErc20Method(banktypes.BalanceOf),
			returnData: balanceReturn,
			query: func(keeper Erc20Keeper, ctx sdk.Context) error {
				_, err := keeper.QueryBalanceOf(ctx, contract, account)
				return err
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := sdk.Context{}.
				WithContext(context.Background()).
				WithEventManager(sdk.NewEventManager()).
				WithGasMeter(storetypes.NewGasMeter(gasLimit))
			ctx.GasMeter().ConsumeGas(consumedGas, "test setup")

			executor := &recordingERC20EVMExecutor{
				callResponse: &evmtypes.MsgEthereumTxResponse{Ret: tc.returnData},
			}
			keeper := Erc20Keeper{ak: queryAccountKeeper{}, ek: executor}

			require.NoError(t, tc.query(keeper, ctx))
			require.Equal(t, 1, executor.callCalls)
			require.NotNil(t, executor.callGasCap)
			require.Zero(t, executor.callGasCap.Cmp(new(big.Int).SetUint64(gasLimit-consumedGas)))
			require.Equal(t, tc.method, executor.callMethod)
			require.False(t, executor.callCommit)
			require.False(t, executor.callFromPrecompile)
		})
	}
}

func TestERC20QueryPropagatesParentLimitedOutOfGas(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
	)

	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit))
	ctx.GasMeter().ConsumeGas(consumedGas, "test setup")

	executor := &recordingERC20EVMExecutor{
		callConsumeGas: gasLimit - consumedGas,
		callErr:        vm.ErrOutOfGas,
	}
	keeper := Erc20Keeper{ak: queryAccountKeeper{}, ek: executor}

	_, err := keeper.QueryTotalSupply(ctx, common.HexToAddress("0x2000"))
	require.ErrorContains(t, err, vm.ErrOutOfGas.Error())
	require.Equal(t, gasLimit, ctx.GasMeter().GasConsumed())
	require.True(t, ctx.GasMeter().IsOutOfGas())
}

func TestERC20QueryDoesNotDoubleChargeEVMGas(t *testing.T) {
	const (
		gasLimit    = uint64(100_000)
		consumedGas = uint64(12_345)
		evmGasUsed  = uint64(1_234)
	)

	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(gasLimit))
	ctx.GasMeter().ConsumeGas(consumedGas, "test setup")
	returnData, err := ABI.Methods[banktypes.GetErc20Method(banktypes.TotalSupply)].Outputs.Pack(big.NewInt(42))
	require.NoError(t, err)

	executor := &recordingERC20EVMExecutor{
		callConsumeGas: evmGasUsed,
		callResponse:   &evmtypes.MsgEthereumTxResponse{GasUsed: evmGasUsed, Ret: returnData},
	}
	keeper := Erc20Keeper{ak: queryAccountKeeper{}, ek: executor}

	_, err = keeper.QueryTotalSupply(ctx, common.HexToAddress("0x2000"))
	require.NoError(t, err)
	require.Equal(t, consumedGas+evmGasUsed, ctx.GasMeter().GasConsumed())
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
