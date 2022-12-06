package keeper

import (
	"encoding/json"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/evmos/ethermint/server/config"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

func (k Keeper) callEVM(
	ctx sdk.Context,
	sender sdk.AccAddress,
	contract *common.Address,
	data []byte,
	amount sdk.Int,
	commit bool,
) (*evmtypes.MsgEthereumTxResponse, uint64, error) {
	gasCap := config.DefaultGasCap

	nonce, err := k.accountKeeper.GetSequence(ctx, sender)
	if err != nil {
		return nil, gasCap, err
	}

	from := common.BytesToAddress(sender.Bytes())

	if commit {
		args, err := json.Marshal(evmtypes.TransactionArgs{
			From: &from,
			To:   contract,
			Data: (*hexutil.Bytes)(&data),
		})
		if err != nil {
			return nil, gasCap, sdkerrors.Wrapf(sdkerrors.ErrJSONMarshal, "failed to marshal tx args: %s", err.Error())
		}

		gasRes, err := k.evmKeeper.EstimateGas(sdk.WrapSDKContext(ctx), &evmtypes.EthCallRequest{
			Args:   args,
			GasCap: config.DefaultGasCap,
		})
		if err != nil {
			return nil, gasCap, err
		}
		gasCap = gasRes.Gas
	}

	msg := ethtypes.NewMessage(
		from,
		contract,
		nonce,
		amount.BigInt(), // amount
		gasCap,          // gasLimit
		big.NewInt(0),   // gasFeeCap
		big.NewInt(0),   // gasTipCap
		big.NewInt(0),   // gasPrice
		data,
		ethtypes.AccessList{}, // AccessList
		!commit,               // isFake
	)

	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), commit)
	if err != nil {
		return nil, gasCap, sdkerrors.Wrap(err, "failed to apply ethereum core message")
	}

	if res.Failed() {
		return nil, gasCap, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res, gasCap, nil
}
