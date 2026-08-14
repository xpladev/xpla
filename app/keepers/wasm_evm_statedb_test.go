package keepers

import (
	"context"
	"errors"
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/statedb"

	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

func TestWasmEVMStateDBAtomicMessengerPassesThroughWithoutSharedStateDB(t *testing.T) {
	expectedErr := errors.New("dispatch failed")
	called := false
	next := wasmkeeper.MessageHandlerFunc(func(
		_ sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		called = true
		return nil, nil, nil, expectedErr
	})

	ctx := newWasmMessengerTestContext()
	_, _, _, err := newWasmEVMStateDBAtomicMessenger(next).DispatchMsg(
		ctx,
		nil,
		"",
		wasmvmtypes.CosmosMsg{},
	)

	require.True(t, called)
	require.ErrorIs(t, err, expectedErr)
}

func TestWasmEVMStateDBAtomicMessengerRollsBackFailedStateAndLogs(t *testing.T) {
	expectedErr := errors.New("dispatch failed")
	ctx, stateDB := newWasmMessengerTestStateDB()
	outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
	failedLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
	stateDB.AddRefund(3)
	stateDB.AddLog(outerLog)

	next := wasmkeeper.MessageHandlerFunc(func(
		ctx sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		ctx.GasMeter().ConsumeGas(11, "failed dispatch")
		stateDB.AddRefund(5)
		stateDB.AddLog(failedLog)
		return nil, nil, nil, expectedErr
	})

	_, _, _, err := newWasmEVMStateDBAtomicMessenger(next).DispatchMsg(
		ctx,
		nil,
		"",
		wasmvmtypes.CosmosMsg{},
	)

	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, uint64(3), stateDB.GetRefund())
	require.Equal(t, []*ethtypes.Log{outerLog}, stateDB.Logs())
	require.Equal(t, uint64(11), ctx.GasMeter().GasConsumed())
}

func TestWasmEVMStateDBAtomicMessengerPreservesEarlierSuccessfulDispatch(t *testing.T) {
	expectedErr := errors.New("second dispatch failed")
	ctx, stateDB := newWasmMessengerTestStateDB()
	outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
	successLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
	failedLog := &ethtypes.Log{Address: common.HexToAddress("0x3000")}
	stateDB.AddRefund(3)
	stateDB.AddLog(outerLog)

	calls := 0
	next := wasmkeeper.MessageHandlerFunc(func(
		_ sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		calls++
		if calls == 1 {
			stateDB.AddRefund(5)
			stateDB.AddLog(successLog)
			return nil, nil, nil, nil
		}

		stateDB.AddRefund(7)
		stateDB.AddLog(failedLog)
		return nil, nil, nil, expectedErr
	})
	messenger := newWasmEVMStateDBAtomicMessenger(next)

	_, _, _, err := messenger.DispatchMsg(ctx, nil, "", wasmvmtypes.CosmosMsg{})
	require.NoError(t, err)
	_, _, _, err = messenger.DispatchMsg(ctx, nil, "", wasmvmtypes.CosmosMsg{})

	require.ErrorIs(t, err, expectedErr)
	require.Equal(t, uint64(8), stateDB.GetRefund())
	require.Equal(t, []*ethtypes.Log{outerLog, successLog}, stateDB.Logs())
}

func TestWasmEVMStateDBAtomicMessengerRollsBackPanicsWithoutRefundingGas(t *testing.T) {
	tests := []struct {
		name       string
		panicValue any
	}{
		{name: "panic", panicValue: &struct{ message string }{message: "boom"}},
		{name: "out of gas", panicValue: storetypes.ErrorOutOfGas{Descriptor: "test"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, stateDB := newWasmMessengerTestStateDB()
			outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
			failedLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
			stateDB.AddRefund(3)
			stateDB.AddLog(outerLog)

			next := wasmkeeper.MessageHandlerFunc(func(
				ctx sdk.Context,
				_ sdk.AccAddress,
				_ string,
				_ wasmvmtypes.CosmosMsg,
			) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
				ctx.GasMeter().ConsumeGas(11, "panicking dispatch")
				stateDB.AddRefund(5)
				stateDB.AddLog(failedLog)
				panic(tc.panicValue)
			})

			var recovered any
			func() {
				defer func() {
					recovered = recover()
				}()
				_, _, _, _ = newWasmEVMStateDBAtomicMessenger(next).DispatchMsg(
					ctx,
					nil,
					"",
					wasmvmtypes.CosmosMsg{},
				)
			}()

			if tc.name == "panic" {
				require.Same(t, tc.panicValue, recovered)
			} else {
				require.Equal(t, tc.panicValue, recovered)
			}
			require.Equal(t, uint64(3), stateDB.GetRefund())
			require.Equal(t, []*ethtypes.Log{outerLog}, stateDB.Logs())
			require.Equal(t, uint64(11), ctx.GasMeter().GasConsumed())
		})
	}
}

func newWasmMessengerTestContext() sdk.Context {
	return sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(1_000_000))
}

func newWasmMessengerTestStateDB() (sdk.Context, *statedb.StateDB) {
	ctx := newWasmMessengerTestContext()
	stateDB := statedb.New(ctx, nil, statedb.TxConfig{})
	return xbanktypes.WithEVMStateDB(ctx, stateDB), stateDB
}
