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
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdktestutil "github.com/cosmos/cosmos-sdk/testutil"
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
	ctx, stateDB, key := newWasmMessengerTestStateDB(t)
	outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
	failedLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
	stateDB.AddRefund(3)
	stateDB.AddLog(outerLog)
	gasBefore := ctx.GasMeter().GasConsumed()

	next := wasmkeeper.MessageHandlerFunc(func(
		ctx sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		ctx.GasMeter().ConsumeGas(11, "failed dispatch")
		ctx.KVStore(key).Set([]byte("failed"), []byte("value"))
		ctx.EventManager().EmitEvent(sdk.NewEvent("failed"))
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
	require.GreaterOrEqual(t, ctx.GasMeter().GasConsumed()-gasBefore, uint64(11))
	require.Nil(t, ctx.KVStore(key).Get([]byte("failed")))
	require.Empty(t, ctx.EventManager().Events())
}

func TestWasmEVMStateDBAtomicMessengerPreservesEarlierSuccessfulDispatch(t *testing.T) {
	expectedErr := errors.New("second dispatch failed")
	ctx, stateDB, key := newWasmMessengerTestStateDB(t)
	outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
	successLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
	failedLog := &ethtypes.Log{Address: common.HexToAddress("0x3000")}
	stateDB.AddRefund(3)
	stateDB.AddLog(outerLog)

	calls := 0
	next := wasmkeeper.MessageHandlerFunc(func(
		dispatchCtx sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		calls++
		if calls == 1 {
			dispatchCtx.KVStore(key).Set([]byte("shared"), []byte("sibling"))
			dispatchCtx.EventManager().EmitEvent(sdk.NewEvent("sibling"))
			stateDB.AddRefund(5)
			stateDB.AddLog(successLog)
			return nil, nil, nil, nil
		}

		dispatchCtx.KVStore(key).Set([]byte("shared"), []byte("failed"))
		dispatchCtx.KVStore(key).Set([]byte("failed"), []byte("value"))
		dispatchCtx.EventManager().EmitEvent(sdk.NewEvent("failed"))
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
	require.Equal(t, []byte("sibling"), ctx.KVStore(key).Get([]byte("shared")))
	require.Nil(t, ctx.KVStore(key).Get([]byte("failed")))
	require.Equal(t, sdk.Events{sdk.NewEvent("sibling")}, ctx.EventManager().Events())
}

func TestWasmEVMStateDBAtomicMessengerNestedSuccessRevertsWithEnclosingFailure(t *testing.T) {
	ctx, stateDB, key := newWasmMessengerTestStateDB(t)
	outerCtx, outerScope, err := stateDB.NewNativeScope(ctx)
	require.NoError(t, err)
	defer outerScope.Rollback()
	outerCtx = xbanktypes.WithEVMStateDB(outerCtx, stateDB)

	childLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
	returnedEvents := []sdk.Event{sdk.NewEvent("returned")}
	gasBefore := ctx.GasMeter().GasConsumed()
	next := wasmkeeper.MessageHandlerFunc(func(
		dispatchCtx sdk.Context,
		_ sdk.AccAddress,
		_ string,
		_ wasmvmtypes.CosmosMsg,
	) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
		dispatchCtx.GasMeter().ConsumeGas(13, "successful nested dispatch")
		dispatchCtx.KVStore(key).Set([]byte("child"), []byte("value"))
		dispatchCtx.EventManager().EmitEvent(sdk.NewEvent("child"))
		stateDB.AddRefund(9)
		stateDB.AddLog(childLog)
		return returnedEvents, nil, nil, nil
	})

	events, _, _, err := newWasmEVMStateDBAtomicMessenger(next).DispatchMsg(
		outerCtx,
		nil,
		"",
		wasmvmtypes.CosmosMsg{},
	)
	require.NoError(t, err)
	require.Equal(t, returnedEvents, events)
	require.Equal(t, []byte("value"), outerCtx.KVStore(key).Get([]byte("child")))
	require.Equal(t, sdk.Events{sdk.NewEvent("child")}, outerCtx.EventManager().Events())
	require.Equal(t, []*ethtypes.Log{childLog}, stateDB.Logs())

	outerScope.Rollback()
	require.Nil(t, ctx.KVStore(key).Get([]byte("child")))
	require.Empty(t, ctx.EventManager().Events())
	require.Empty(t, stateDB.Logs())
	require.Zero(t, stateDB.GetRefund())
	require.GreaterOrEqual(t, ctx.GasMeter().GasConsumed()-gasBefore, uint64(13),
		"gas must survive enclosing rollback")
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
			ctx, stateDB, key := newWasmMessengerTestStateDB(t)
			outerLog := &ethtypes.Log{Address: common.HexToAddress("0x1000")}
			failedLog := &ethtypes.Log{Address: common.HexToAddress("0x2000")}
			stateDB.AddRefund(3)
			stateDB.AddLog(outerLog)
			gasBefore := ctx.GasMeter().GasConsumed()

			next := wasmkeeper.MessageHandlerFunc(func(
				ctx sdk.Context,
				_ sdk.AccAddress,
				_ string,
				_ wasmvmtypes.CosmosMsg,
			) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
				ctx.GasMeter().ConsumeGas(11, "panicking dispatch")
				ctx.KVStore(key).Set([]byte("failed"), []byte("value"))
				ctx.EventManager().EmitEvent(sdk.NewEvent("failed"))
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
			require.GreaterOrEqual(t, ctx.GasMeter().GasConsumed()-gasBefore, uint64(11))
			require.Nil(t, ctx.KVStore(key).Get([]byte("failed")))
			require.Empty(t, ctx.EventManager().Events())
		})
	}
}

func newWasmMessengerTestContext() sdk.Context {
	return sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(1_000_000))
}

func newWasmMessengerTestStateDB(t *testing.T) (sdk.Context, *statedb.StateDB, *storetypes.KVStoreKey) {
	t.Helper()
	key := storetypes.NewKVStoreKey("wasm_evm_statedb")
	ctx := sdktestutil.DefaultContext(key, storetypes.NewTransientStoreKey("wasm_evm_statedb_t")).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(1_000_000))
	keeper := wasmMessengerTestKeeper{key: key}
	stateDB := statedb.New(ctx, keeper, statedb.TxConfig{})
	actionCtx, endAction, err := stateDB.BeginNativeAction()
	require.NoError(t, err)
	t.Cleanup(endAction)
	return xbanktypes.WithEVMStateDB(actionCtx, stateDB), stateDB, key
}

type wasmMessengerTestKeeper struct {
	key *storetypes.KVStoreKey
}

func (wasmMessengerTestKeeper) GetAccount(sdk.Context, common.Address) *statedb.Account {
	return nil
}

func (wasmMessengerTestKeeper) GetBalance(sdk.Context, common.Address) *uint256.Int {
	return new(uint256.Int)
}

func (wasmMessengerTestKeeper) GetState(sdk.Context, common.Address, common.Hash) common.Hash {
	return common.Hash{}
}

func (wasmMessengerTestKeeper) GetCode(sdk.Context, common.Hash) []byte { return nil }

func (wasmMessengerTestKeeper) GetCodeHash(sdk.Context, common.Address) common.Hash {
	return common.Hash{}
}

func (wasmMessengerTestKeeper) ForEachStorage(
	sdk.Context,
	common.Address,
	func(common.Hash, common.Hash) bool,
) {
}

func (wasmMessengerTestKeeper) SetAccount(sdk.Context, common.Address, statedb.Account) error {
	return nil
}

func (wasmMessengerTestKeeper) DeleteState(sdk.Context, common.Address, common.Hash) {}

func (wasmMessengerTestKeeper) SetState(sdk.Context, common.Address, common.Hash, []byte) {}

func (wasmMessengerTestKeeper) DeleteCode(sdk.Context, []byte) {}

func (wasmMessengerTestKeeper) SetCode(sdk.Context, []byte, []byte) {}

func (wasmMessengerTestKeeper) DeleteAccount(sdk.Context, common.Address) error { return nil }

func (k wasmMessengerTestKeeper) KVStoreKeys() map[string]*storetypes.KVStoreKey {
	return map[string]*storetypes.KVStoreKey{"test": k.key}
}
