package wasm_test

import (
	"context"
	"encoding/binary"
	"errors"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/precompile/wasm"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

// Control only the native action; use the real keepers, Wasm entry points,
// upstream wrapper and StateDB to observe exact settlement and cache boundaries.
type gasRecordingWasmServer struct {
	wasm.WasmMsgServer
	execute func(context.Context, *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error)
}

func setTestContractMetadata(t *testing.T, input testutil.TestInput, address sdk.AccAddress) {
	t.Helper()
	key := input.App.EvmKeeper.KVStoreKeys()[wasmtypes.StoreKey]
	input.Ctx.KVStore(key).Set(wasmtypes.GetContractAddressKey(address), input.App.AppCodec().MustMarshal(&wasmtypes.ContractInfo{}))
}

func (s gasRecordingWasmServer) ExecuteContract(ctx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	return s.execute(ctx, msg)
}

func TestWasmEntryPointGasSettlement(t *testing.T) {
	tests := []struct {
		name      string
		actionErr error
		wantErr   error
	}{
		{name: "success"},
		{
			name:      "ordinary error",
			actionErr: errors.New("native action failed"),
			wantErr:   vm.ErrExecutionReverted,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			input := testutil.CreateTestInput(t)
			sender := common.HexToAddress("0x1234")
			target := common.HexToAddress("0x5678")
			wasmAddress := sdk.AccAddress(target.Bytes())
			account := input.AccountKeeper.NewAccountWithAddress(input.Ctx, wasmAddress)
			input.AccountKeeper.SetAccount(input.Ctx, account)
			setTestContractMetadata(t, input, wasmAddress)
			const budget, initialGas = uint64(1_000_000), uint64(30)
			ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).WithEventManager(sdk.NewEventManager())
			ctx.GasMeter().ConsumeGas(initialGas, "before native wrapper")
			db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
			evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
			evm.Origin = common.HexToAddress("0x9999")
			key := input.App.EvmKeeper.KVStoreKeys()[wasmtypes.StoreKey]
			var actionCtx sdk.Context
			calls := 0
			server := gasRecordingWasmServer{execute: func(goCtx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
				actionCtx = sdk.UnwrapSDKContext(goCtx)
				calls++
				require.Equal(t, sdk.AccAddress(sender.Bytes()).String(), msg.Sender)
				sharedDB, found := xbanktypes.EVMStateDBFromContext(actionCtx)
				require.True(t, found, "the production entry point must inject the StateDB")
				require.Same(t, db, sharedDB)
				actionCtx.KVStore(key).Set([]byte("gas-settlement"), []byte("written"))
				actionCtx.EventManager().EmitEvent(sdk.NewEvent("native_action"))
				actionCtx.GasMeter().ConsumeGas(100, "native work")
				if tc.actionErr != nil {
					return nil, tc.actionErr
				}
				return &wasmtypes.MsgExecuteContractResponse{Data: []byte("native return data")}, nil
			}}
			p := wasm.NewPrecompiledWasm(server, input.App.WasmKeeper, input.BankKeeper)
			contract := vm.NewContract(sender, wasm.Address, uint256.NewInt(0), budget, nil)
			var err error
			contract.Input, err = wasm.ABI.Pack("executeContract", sender, target, []byte(`{"increment":{}}`), cmn.NewCoinsResponse(nil))
			require.NoError(t, err)
			snapshot := db.Snapshot()
			bz, err := p.Run(evm, contract, false)
			require.Equal(t, 1, calls)
			consumed := actionCtx.GasMeter().GasConsumed()
			require.Equal(t, budget-consumed, contract.Gas, "the SDK delta must be settled exactly once")
			cacheCtx, cacheErr := db.GetCacheContext()
			require.NoError(t, cacheErr)
			if tc.wantErr == nil {
				require.NoError(t, err)
				values, unpackErr := wasm.ABI.Unpack("executeContract", bz)
				require.NoError(t, unpackErr)
				require.Equal(t, []byte("native return data"), values[0])
				require.Equal(t, []byte("written"), cacheCtx.KVStore(key).Get([]byte("gas-settlement")))
				require.Equal(t, sdk.Events{sdk.NewEvent("native_action")}, cacheCtx.EventManager().Events())
				require.Len(t, db.Logs(), 1)
			} else {
				require.Nil(t, cacheCtx.KVStore(key).Get([]byte("gas-settlement")))
				require.Empty(t, cacheCtx.EventManager().Events())
				require.Empty(t, db.Logs())
				require.Same(t, tc.wantErr, err)
				if tc.actionErr != nil {
					reason, unpackErr := abi.UnpackRevert(bz)
					require.NoError(t, unpackErr)
					require.Equal(t, tc.actionErr.Error(), reason)
				} else {
					require.Nil(t, bz)
				}
			}
			db.RevertToSnapshot(snapshot)
			require.Nil(t, cacheCtx.KVStore(key).Get([]byte("gas-settlement")))
			require.Empty(t, cacheCtx.EventManager().Events())
			require.Empty(t, db.Logs())
		})
	}
}

func TestWasmEntryPointOutOfGasThroughEVM(t *testing.T) {
	t.Run("failed frame returns no gas", func(t *testing.T) {
		fixture := newWasmOutOfGasFixture(t, false)
		ret, left, err := fixture.evm.Call(
			fixture.sender,
			wasm.Address,
			fixture.input,
			fixture.childGas,
			uint256.NewInt(0),
		)
		require.ErrorIs(t, err, vm.ErrOutOfGas)
		require.Nil(t, ret)
		require.Zero(t, left, "the failed EVM child frame must return no gas")
		fixture.requireOutOfGasRollback(t)
	})

	t.Run("caught failure preserves outer reserve", func(t *testing.T) {
		fixture := newWasmOutOfGasFixture(t, true)
		fixture.db.SetCode(fixture.outer, wasmChildCallCode(
			fixture.precompileAddress,
			fixture.input,
			fixture.childGas,
		))

		ret, left, err := fixture.evm.Call(
			fixture.sender,
			fixture.outer,
			nil,
			fixture.outerGas,
			uint256.NewInt(0),
		)

		require.NoError(t, err)
		require.Len(t, ret, 32)
		require.Zero(t, new(big.Int).SetBytes(ret).Sign(), "CALL must report the child failure")
		require.Positive(t, left, "the outer frame must retain gas and continue")
		require.Less(t, left, fixture.outerGas)
		require.ErrorIs(t, fixture.childErr, vm.ErrOutOfGas)
		require.Equal(t, fixture.childGas, fixture.childGasUsed,
			"the failed child must consume exactly its forwarded gas")
		fixture.requireOutOfGasRollback(t)
	})
}

type wasmOutOfGasFixture struct {
	db                *statedb.StateDB
	evm               *vm.EVM
	actionCtx         sdk.Context
	key               *storetypes.KVStoreKey
	sender            common.Address
	outer             common.Address
	marker            common.Address
	markerSlot        common.Hash
	markerValue       common.Hash
	precompileAddress common.Address
	input             []byte
	childGas          uint64
	nativeGas         uint64
	outerGas          uint64
	childDepth        int
	childErr          error
	childGasUsed      uint64
	calls             int
}

func newWasmOutOfGasFixture(t *testing.T, caught bool) *wasmOutOfGasFixture {
	t.Helper()
	input := testutil.CreateTestInput(t)
	sender := common.HexToAddress("0x1234")
	outer := common.HexToAddress("0x9999")
	target := common.HexToAddress("0x5678")
	wasmAddress := sdk.AccAddress(target.Bytes())
	account := input.AccountKeeper.NewAccountWithAddress(input.Ctx, wasmAddress)
	input.AccountKeeper.SetAccount(input.Ctx, account)
	setTestContractMetadata(t, input, wasmAddress)

	const (
		childGas = uint64(1_000_000)
		outerGas = uint64(2_000_000)
	)
	ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(outerGas)).WithEventManager(sdk.NewEventManager())
	ctx.GasMeter().ConsumeGas(30, "before native wrapper")
	db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
	fixture := &wasmOutOfGasFixture{
		db:          db,
		key:         input.App.EvmKeeper.KVStoreKeys()[wasmtypes.StoreKey],
		sender:      sender,
		outer:       outer,
		marker:      common.HexToAddress("0x7777"),
		markerSlot:  common.HexToHash("0x01"),
		markerValue: common.HexToHash("0x02"),
		childGas:    childGas,
		outerGas:    outerGas,
		childDepth:  -1,
	}

	expectedSender := sender
	fixture.precompileAddress = wasm.Address
	if caught {
		expectedSender = outer
	}
	triggerOutOfGas := func(goCtx context.Context) {
		fixture.actionCtx = sdk.UnwrapSDKContext(goCtx)
		fixture.calls++
		sharedDB, found := xbanktypes.EVMStateDBFromContext(fixture.actionCtx)
		require.True(t, found)
		require.Same(t, db, sharedDB)
		require.Equal(t, fixture.nativeGas, fixture.actionCtx.GasMeter().Limit(),
			"each EVM child frame must receive a fresh SDK gas meter")

		fixture.actionCtx.KVStore(fixture.key).Set([]byte("oog-sdk-write"), []byte("written"))
		fixture.actionCtx.EventManager().EmitEvent(sdk.NewEvent("oog_native_action"))
		db.SetState(fixture.marker, fixture.markerSlot, fixture.markerValue)
		db.AddLog(&ethtypes.Log{Address: fixture.marker})
		fixture.actionCtx.GasMeter().ConsumeGas(100, "native work before OOG")
		fixture.actionCtx.GasMeter().ConsumeGas(fixture.actionCtx.GasMeter().GasRemaining()+1, "native OOG")
	}
	server := gasRecordingWasmServer{execute: func(goCtx context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
		require.Equal(t, sdk.AccAddress(expectedSender.Bytes()).String(), msg.Sender)
		triggerOutOfGas(goCtx)
		return nil, nil
	}}
	p := wasm.NewPrecompiledWasm(server, input.App.WasmKeeper, input.BankKeeper)
	precompiles := vm.PrecompiledContracts{wasm.Address: p}

	tracer := &tracing.Hooks{
		OnEnter: func(depth int, _ byte, _, to common.Address, _ []byte, _ uint64, _ *big.Int) {
			if to == fixture.precompileAddress {
				fixture.childDepth = depth
				fixture.childErr = nil
			}
		},
		OnExit: func(depth int, _ []byte, gasUsed uint64, err error, _ bool) {
			if fixture.childDepth == depth {
				fixture.childErr = err
				fixture.childGasUsed = gasUsed
				fixture.childDepth = -1
			}
		},
	}
	fixture.evm = vm.NewEVM(vm.BlockContext{
		BlockNumber: big.NewInt(1),
		Time:        1,
		Difficulty:  new(big.Int),
		BaseFee:     new(big.Int),
		GasLimit:    outerGas,
		CanTransfer: func(vm.StateDB, common.Address, *uint256.Int) bool { return true },
		Transfer:    func(vm.StateDB, common.Address, common.Address, *uint256.Int) {},
	}, db, vmtypes.GetEthChainConfig(), vm.Config{Tracer: tracer})
	fixture.evm.Origin = sender
	fixture.evm.SetPrecompiles(precompiles)

	var err error
	fixture.input, err = wasm.ABI.Pack(
		"executeContract",
		expectedSender,
		target,
		[]byte(`{"increment":{}}`),
		cmn.NewCoinsResponse(nil),
	)
	require.NoError(t, err)
	requiredGas := p.RequiredGas(fixture.input)
	require.Less(t, requiredGas, childGas)
	fixture.nativeGas = childGas - requiredGas
	return fixture
}

func (f *wasmOutOfGasFixture) requireOutOfGasRollback(t *testing.T) {
	t.Helper()
	require.Equal(t, 1, f.calls, "the native OOG action must be reached exactly once")
	require.NotNil(t, f.actionCtx.GasMeter())
	cacheCtx, err := f.db.GetCacheContext()
	require.NoError(t, err)
	require.Nil(t, cacheCtx.KVStore(f.key).Get([]byte("oog-sdk-write")))
	require.Empty(t, cacheCtx.EventManager().Events())
	require.Equal(t, common.Hash{}, f.db.GetState(f.marker, f.markerSlot))
	require.Empty(t, f.db.Logs())
}

func wasmChildCallCode(target common.Address, input []byte, gas uint64) []byte {
	code := []byte{
		byte(vm.PUSH2), byte(len(input) >> 8), byte(len(input)),
		byte(vm.PUSH2), 0, 0,
		byte(vm.PUSH1), 0,
		byte(vm.CODECOPY),
		byte(vm.PUSH1), 0,
		byte(vm.PUSH1), 0,
		byte(vm.PUSH2), byte(len(input) >> 8), byte(len(input)),
		byte(vm.PUSH1), 0,
	}
	code = append(code, byte(vm.PUSH1), 0)
	code = append(code, byte(vm.PUSH20))
	code = append(code, target.Bytes()...)
	var gasBytes [8]byte
	binary.BigEndian.PutUint64(gasBytes[:], gas)
	code = append(code, byte(vm.PUSH8))
	code = append(code, gasBytes[:]...)
	code = append(code, byte(vm.CALL))
	code = append(code,
		byte(vm.PUSH1), 0, byte(vm.MSTORE),
		byte(vm.PUSH1), 32, byte(vm.PUSH1), 0, byte(vm.RETURN),
	)
	binary.BigEndian.PutUint16(code[4:6], uint16(len(code)))
	return append(code, input...)
}
