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
	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())

	stateDB := statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	res, err := k.ek.CallEVM(ctx, stateDB, ABI, moduleAddress, contractAddress, false, false, nil, types.GetErc20Method(types.TotalSupply))
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

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
	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())
	ethAccount := common.BytesToAddress(account.Bytes())

	stateDB := statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	res, err := k.ek.CallEVM(ctx, stateDB, ABI, moduleAddress, contractAddress, false, false, nil, types.GetErc20Method(types.BalanceOf), ethAccount)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

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

func (k Erc20Keeper) ExecuteTransfer(ctx sdk.Context, contractAddress common.Address, sender, to sdk.AccAddress, amount *big.Int) error {
	stateDB, callFromPrecompile := types.EVMStateDBFromContext(ctx)
	if !callFromPrecompile {
		stateDB = statedb.New(ctx, k.ek, statedb.NewEmptyTxConfig())
	}

	ethSender := common.BytesToAddress(sender.Bytes())
	ethTo := common.BytesToAddress(to.Bytes())
	method := types.GetErc20Method(types.Transfer)

	var (
		res *evmtypes.MsgEthereumTxResponse
		err error
	)

	if !callFromPrecompile {
		res, err = k.ek.CallEVM(ctx, stateDB, ABI, ethSender, contractAddress, true, false, nil, method, ethTo, amount)
	} else {
		var data []byte
		data, err = ABI.Pack(method, ethTo, amount)
		if err != nil {
			return errorsmod.Wrap(
				evmtypes.ErrABIPack,
				errorsmod.Wrap(err, "failed to create transaction data").Error(),
			)
		}

		msg := core.Message{
			From:       ethSender,
			To:         &contractAddress,
			Nonce:      k.ek.GetNonce(ctx, ethSender),
			Value:      big.NewInt(0),
			GasLimit:   ctx.GasMeter().GasRemaining(),
			GasPrice:   big.NewInt(0),
			GasTipCap:  big.NewInt(0),
			GasFeeCap:  big.NewInt(0),
			Data:       data,
			AccessList: ethtypes.AccessList{},
		}

		res, err = k.ek.ApplyMessage(ctx, stateDB, msg, nil, false, true, true)
		if err == nil && res.Failed() {
			// A nested VM failure consumes the caller's full remaining gas budget.
			gasMeter := ctx.GasMeter()
			gasMeter.RefundGas(gasMeter.GasConsumed(), "reset the gas count")
			gasMeter.ConsumeGas(gasMeter.Limit(), "apply evm transaction")
			return errorsmod.Wrapf(
				errorsmod.Wrap(evmtypes.ErrVMExecution, res.VmError),
				"contract call failed: method '%s', contract '%s'",
				method,
				contractAddress,
			)
		}
		if err == nil {
			ctx.GasMeter().ConsumeGas(res.GasUsed, "apply evm message")
		}
	}

	if err != nil {
		if callFromPrecompile {
			return errorsmod.Wrapf(err, "contract call failed: method '%s', contract '%s'", method, contractAddress)
		}
		return err
	}

	unpacked, err := ABI.Unpack(method, res.Return())
	if err != nil {
		return err
	}

	if len(unpacked) == 0 || !unpacked[0].(bool) {
		return types.ErrErc20Transfer
	}

	return nil
}
