package keeper

import (
	"embed"
	"encoding/json"
	"math/big"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/cosmos/evm/server/config"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	precompileutil "github.com/xpladev/xpla/precompile/util"
	"github.com/xpladev/xpla/x/bank/types"
)

const abiFile = "IERC20.abi"

var (
	ABI = abi.ABI{}

	//go:embed IERC20.abi
	abiFS embed.FS
)

type Erc20Keeper struct {
	ek types.EvmKeeper
}

func init() {
	var err error
	ABI, err = precompileutil.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewErc20Keeper(ek types.EvmKeeper) Erc20Keeper {
	return Erc20Keeper{
		ek: ek,
	}
}

func (k Erc20Keeper) QueryTotalSupply(ctx sdk.Context, contractAddress common.Address) (sdkmath.Int, error) {
	moduleAddress := common.BytesToAddress(authtypes.NewModuleAddress(banktypes.ModuleName).Bytes())

	data, err := ABI.Pack(types.GetErc20Method(types.TotalSupply))
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	res, err := k.callEVM(ctx, moduleAddress, &contractAddress, false, data)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.TotalSupply), res)
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
	moduleAddress := common.BytesToAddress(authtypes.NewModuleAddress(banktypes.ModuleName).Bytes())
	ethAccount := common.BytesToAddress(account.Bytes())

	data, err := ABI.Pack(types.GetErc20Method(types.BalanceOf), ethAccount)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	res, err := k.callEVM(ctx, moduleAddress, &contractAddress, false, data)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.BalanceOf), res)
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
	ethSender := common.BytesToAddress(sender.Bytes())
	ethTo := common.BytesToAddress(to.Bytes())

	data, err := ABI.Pack(types.GetErc20Method(types.Transfer), ethTo, amount)
	if err != nil {
		return err
	}

	res, err := k.callEVM(ctx, ethSender, &contractAddress, true, data)
	if err != nil {
		return err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.Transfer), res)
	if err != nil {
		return err
	}

	if len(unpacked) == 0 || !unpacked[0].(bool) {
		return types.ErrErc20Transfer
	}

	return nil
}

func (bek Erc20Keeper) callEVM(
	ctx sdk.Context,
	from common.Address,
	contract *common.Address,
	commit bool,
	data []byte,
) ([]byte, error) {
	nonce := bek.ek.GetNonce(ctx, from)

	gasCap := config.DefaultGasCap
	if commit {
		args, err := json.Marshal(evmtypes.TransactionArgs{
			From: &from,
			To:   contract,
			Data: (*hexutil.Bytes)(&data),
		})
		if err != nil {
			return nil, errorsmod.Wrapf(errortypes.ErrJSONMarshal, "failed to marshal tx args: %s", err.Error())
		}

		gasRes, err := bek.ek.EstimateGas(ctx, &evmtypes.EthCallRequest{
			Args:   args,
			GasCap: config.DefaultGasCap,
		})
		if err != nil {
			return nil, err
		}
		gasCap = gasRes.Gas
	}

	msg := core.Message{
		From:       from,
		To:         contract,
		Nonce:      nonce,
		Value:      big.NewInt(0),
		GasLimit:   gasCap,
		GasPrice:   big.NewInt(0),
		GasTipCap:  big.NewInt(0),
		GasFeeCap:  big.NewInt(0),
		Data:       data,
		AccessList: ethtypes.AccessList{},
	}

	res, err := bek.ek.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), true, false)
	if err != nil {
		return nil, err
	}

	if res.Failed() {
		return nil, errorsmod.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res.Ret, nil
}
