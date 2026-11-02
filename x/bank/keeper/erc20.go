package keeper

import (
	"bytes"
	"math/big"

	_ "embed"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmconfig "github.com/cosmos/evm/server/config"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/xpladev/xpla/x/bank/types"
)

var (
	ABI = abi.ABI{}

	//go:embed IERC20.json
	f []byte
)

type Erc20Keeper struct {
	ak banktypes.AccountKeeper
	ek types.EvmKeeper
}

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

func NewErc20Keeper(ak banktypes.AccountKeeper, ek types.EvmKeeper) Erc20Keeper {
	return Erc20Keeper{
		ak: ak,
		ek: ek,
	}
}

func (k Erc20Keeper) QueryTotalSupply(ctx sdk.Context, contractAddress common.Address) (sdkmath.Int, error) {
	stateDB, callFromPrecompile := types.EVMStateDBFromContext(ctx)
	if !callFromPrecompile {
		stateDB = statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	} else {
		var scope *statedb.NativeScope
		var err error
		ctx, scope, err = stateDB.NewNativeScope(ctx)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		defer scope.Rollback()
		if err := stateDB.ReconcileNativeBalances(ctx); err != nil {
			return sdkmath.ZeroInt(), err
		}
	}

	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())
	gasCap := new(big.Int).SetUint64(ctx.GasMeter().GasRemaining())
	res, err := k.ek.CallEVM(ctx, stateDB, ABI, moduleAddress, contractAddress, false, callFromPrecompile, gasCap, types.GetErc20Method(types.TotalSupply))
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	chargeSharedQueryGas(ctx, res, callFromPrecompile)

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.TotalSupply), res.Return())
	if err != nil || len(unpacked) == 0 {
		return sdkmath.ZeroInt(), err
	}

	bigTotalSupply, ok := unpacked[0].(*big.Int)
	if !ok {
		return sdkmath.ZeroInt(), types.ErrErc20TotalSupply
	}

	totalSupply := sdkmath.NewIntFromBigInt(bigTotalSupply)

	return totalSupply, nil
}

func (k Erc20Keeper) QueryBalanceOf(ctx sdk.Context, contractAddress common.Address, account sdk.AccAddress) (sdkmath.Int, error) {
	stateDB, callFromPrecompile := types.EVMStateDBFromContext(ctx)
	if !callFromPrecompile {
		stateDB = statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	} else {
		var scope *statedb.NativeScope
		var err error
		ctx, scope, err = stateDB.NewNativeScope(ctx)
		if err != nil {
			return sdkmath.ZeroInt(), err
		}
		defer scope.Rollback()
		if err := stateDB.ReconcileNativeBalances(ctx); err != nil {
			return sdkmath.ZeroInt(), err
		}
	}

	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())
	ethAccount := common.BytesToAddress(account.Bytes())
	gasCap := new(big.Int).SetUint64(ctx.GasMeter().GasRemaining())
	res, err := k.ek.CallEVM(ctx, stateDB, ABI, moduleAddress, contractAddress, false, callFromPrecompile, gasCap, types.GetErc20Method(types.BalanceOf), ethAccount)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}
	chargeSharedQueryGas(ctx, res, callFromPrecompile)

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.BalanceOf), res.Return())
	if err != nil || len(unpacked) == 0 {
		return sdkmath.ZeroInt(), err
	}

	bigBalance, ok := unpacked[0].(*big.Int)
	if !ok {
		return sdkmath.ZeroInt(), types.ErrErc20Balance
	}

	balance := sdkmath.NewIntFromBigInt(bigBalance)

	return balance, nil
}

// CallEVM has already charged GasUsed. A shared StateDB can contain refunds
// earned by the outer transaction, but a read-only bridge query must not borrow
// them. Charge only the difference to the query's unreimbursed execution cost;
// the surrounding native scope restores the outer refund counter unchanged.
func chargeSharedQueryGas(ctx sdk.Context, res *evmtypes.MsgEthereumTxResponse, shared bool) {
	if !shared || res.MaxUsedGas <= res.GasUsed {
		return
	}
	ctx.GasMeter().ConsumeGas(res.MaxUsedGas-res.GasUsed, "apply evm query without outer refund")
}

func (k Erc20Keeper) ExecuteTransfer(ctx sdk.Context, contractAddress common.Address, sender, to sdk.AccAddress, amount *big.Int) error {
	stateDB, callFromPrecompile := types.EVMStateDBFromContext(ctx)
	if !callFromPrecompile {
		stateDB = statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	} else {
		var scope *statedb.NativeScope
		var err error
		ctx, scope, err = stateDB.NewNativeScope(ctx)
		if err != nil {
			return err
		}
		defer scope.Rollback()
		if err := stateDB.ReconcileNativeBalances(ctx); err != nil {
			return err
		}

		if err := k.executeTransfer(ctx, stateDB, contractAddress, sender, to, amount, true); err != nil {
			return err
		}
		if err := stateDB.FlushToCacheCtx(); err != nil {
			return err
		}
		return scope.Write()
	}

	return k.executeTransfer(ctx, stateDB, contractAddress, sender, to, amount, false)
}

func (k Erc20Keeper) executeTransfer(
	ctx sdk.Context,
	stateDB *statedb.StateDB,
	contractAddress common.Address,
	sender, to sdk.AccAddress,
	amount *big.Int,
	callFromPrecompile bool,
) error {
	ethSender := common.BytesToAddress(sender.Bytes())
	ethTo := common.BytesToAddress(to.Bytes())
	method := types.GetErc20Method(types.Transfer)

	data, err := ABI.Pack(method, ethTo, amount)
	if err != nil {
		return errorsmod.Wrap(
			evmtypes.ErrABIPack,
			errorsmod.Wrap(err, "failed to create transaction data").Error(),
		)
	}

	gasLimit := evmconfig.DefaultGasCap
	if callFromPrecompile {
		gasLimit = ctx.GasMeter().GasRemaining()
	}
	// Internal transfers can originate from a Wasm contract whose 20-byte suffix
	// has no auth account. Unlike CallEVM's GetSequence, GetNonce permits that
	// absence without resolving the sender to the Wasm account.
	msg := core.Message{
		From:       ethSender,
		To:         &contractAddress,
		Nonce:      k.ek.GetNonce(ctx, ethSender),
		Value:      big.NewInt(0),
		GasLimit:   gasLimit,
		GasPrice:   big.NewInt(0),
		GasTipCap:  big.NewInt(0),
		GasFeeCap:  big.NewInt(0),
		Data:       data,
		AccessList: ethtypes.AccessList{},
	}

	res, err := k.ek.ApplyMessage(ctx, stateDB, msg, nil, !callFromPrecompile, callFromPrecompile, true)
	if err != nil || res.Failed() {
		// Native VM failures and all nested precompile failures consume the full
		// remaining SDK gas budget, matching the existing call paths.
		if callFromPrecompile || err == nil {
			gasMeter := ctx.GasMeter()
			gasMeter.RefundGas(gasMeter.GasConsumed(), "reset the gas count")
			gasMeter.ConsumeGas(gasMeter.Limit(), "apply evm transaction")
		}
		if err == nil {
			err = errorsmod.Wrap(evmtypes.ErrVMExecution, res.VmError)
		}
		return errorsmod.Wrapf(err, "contract call failed: method '%s', contract '%s'", method, contractAddress)
	}
	if callFromPrecompile {
		ctx.GasMeter().ConsumeGas(res.MaxUsedGas, "apply evm message")
	} else {
		ctx.GasMeter().ConsumeGas(res.GasUsed, "apply evm message")
	}

	unpacked, err := ABI.Unpack(method, res.Return())
	if err != nil {
		return err
	}

	if len(unpacked) == 0 {
		return types.ErrErc20Transfer
	}
	transferred, ok := unpacked[0].(bool)
	if !ok || !transferred {
		return types.ErrErc20Transfer
	}

	return nil
}
