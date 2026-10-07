package wasm_test

import (
	"context"
	"errors"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
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

type balanceGasWasmKeeper struct {
	wasm.WasmKeeper
	ctx     sdk.Context
	failure error
	calls   int
}

func (k *balanceGasWasmKeeper) ResolveContractAddress(goCtx context.Context, address sdk.AccAddress) (sdk.AccAddress, error) {
	k.ctx = sdk.UnwrapSDKContext(goCtx)
	k.calls++
	k.ctx.GasMeter().ConsumeGas(100, "balance resolver work")
	if k.failure != nil {
		return nil, k.failure
	}
	return k.WasmKeeper.ResolveContractAddress(goCtx, address)
}

func TestBalanceEntryPointGasSettlement(t *testing.T) {
	for _, tc := range []struct {
		name        string
		delegate    bool
		resolverErr bool
	}{
		{name: "regular/success"},
		{name: "delegate/success", delegate: true},
		{name: "regular/resolver error", resolverErr: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := testutil.CreateTestInput(t)
			sender, target := common.HexToAddress("0x1234"), common.HexToAddress("0x5678")
			address := sdk.AccAddress(target.Bytes())
			require.NoError(t, input.InitAccountWithCoins(address, sdk.NewCoins(sdk.NewInt64Coin("axpla", 42))))
			setTestContractMetadata(t, input, address)
			const budget, initialGas = uint64(1_000_000), uint64(30)
			ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).WithEventManager(sdk.NewEventManager())
			ctx.GasMeter().ConsumeGas(initialGas, "before wrapper")
			db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
			evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
			evm.Origin = sender
			keeper := &balanceGasWasmKeeper{WasmKeeper: input.App.WasmKeeper}
			if tc.resolverErr {
				keeper.failure = errors.New("balance alias lookup failed")
			}
			p := wasm.NewPrecompiledWasm(nil, keeper, input.BankKeeper)
			callAddress := wasm.Address
			if tc.delegate {
				callAddress = wasm.DelegatecallAddress
			}
			contract := vm.NewContract(sender, callAddress, uint256.NewInt(0), budget, nil)
			var err error
			contract.Input, err = wasm.ABI.Pack("balance", target, "axpla")
			require.NoError(t, err)
			var result []byte
			if tc.delegate {
				result, err = p.RunDelegate(evm, contract, true)
			} else {
				result, err = p.Run(evm, contract, true)
			}
			require.Equal(t, 1, keeper.calls)
			sharedDB, found := xbanktypes.EVMStateDBFromContext(keeper.ctx)
			require.True(t, found)
			require.Same(t, db, sharedDB)
			consumed := keeper.ctx.GasMeter().GasConsumed() - initialGas
			require.Positive(t, consumed)
			require.Equal(t, budget-consumed, contract.Gas, "SDK delta must settle exactly once")
			if !tc.resolverErr {
				require.NoError(t, err)
				values, unpackErr := wasm.ABI.Unpack("balance", result)
				require.NoError(t, unpackErr)
				require.Equal(t, big.NewInt(42), values[0])
			} else {
				require.ErrorIs(t, err, vm.ErrExecutionReverted)
				reason, unpackErr := abi.UnpackRevert(result)
				require.NoError(t, unpackErr)
				require.Equal(t, keeper.failure.Error(), reason)
			}
			cacheCtx, cacheErr := db.GetCacheContext()
			require.NoError(t, cacheErr)
			require.Empty(t, cacheCtx.EventManager().Events())
			require.Empty(t, db.Logs())
			require.Equal(t, int64(42), input.BankKeeper.GetBalance(cacheCtx, address, "axpla").Amount.Int64())
		})
	}
}
