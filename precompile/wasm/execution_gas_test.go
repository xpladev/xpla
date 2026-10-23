package wasm_test

import (
	"context"
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
	callModes := []struct {
		name     string
		delegate bool
	}{
		{name: "regular"},
		{name: "delegate", delegate: true},
	}
	tests := []struct {
		name      string
		actionErr error
		outOfGas  bool
		wantErr   error
	}{
		{name: "success"},
		{
			name:      "ordinary error",
			actionErr: errors.New("native action failed"),
			wantErr:   vm.ErrExecutionReverted,
		},
		{
			name:     "SDK OOG panic",
			outOfGas: true,
			wantErr:  vm.ErrOutOfGas,
		},
	}

	for _, mode := range callModes {
		for _, tc := range tests {
			t.Run(mode.name+"/"+tc.name, func(t *testing.T) {
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
				evm.Origin = sender
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
					if tc.outOfGas {
						actionCtx.GasMeter().ConsumeGas(actionCtx.GasMeter().GasRemaining()+1, "native OOG")
					}
					return &wasmtypes.MsgExecuteContractResponse{Data: []byte("native return data")}, nil
				}}
				p := wasm.NewPrecompiledWasm(server, input.App.WasmKeeper, input.BankKeeper)
				caller, address := sender, wasm.Address
				if mode.delegate {
					caller, address = common.HexToAddress("0x9999"), wasm.DelegatecallAddress
				}
				contract := vm.NewContract(caller, address, uint256.NewInt(0), budget, nil)
				var err error
				contract.Input, err = wasm.ABI.Pack("executeContract", sender, target, []byte(`{"increment":{}}`), cmn.NewCoinsResponse(nil))
				require.NoError(t, err)
				snapshot := db.Snapshot()
				var bz []byte
				if mode.delegate {
					bz, err = p.RunDelegate(evm, contract, false)
				} else {
					bz, err = p.Run(evm, contract, false)
				}
				require.Equal(t, 1, calls)
				consumed := actionCtx.GasMeter().GasConsumed() - initialGas
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
						require.Equal(t, uint64(29), contract.Gas, "only upstream HandleGasError settles the panic")
					}
				}
				db.RevertToSnapshot(snapshot)
				require.Nil(t, cacheCtx.KVStore(key).Get([]byte("gas-settlement")))
				require.Empty(t, db.Logs())
			})
		}
	}
}
