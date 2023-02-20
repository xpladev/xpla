package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

func (k Keeper) callEVM(
	ctx sdk.Context,
	sender sdk.AccAddress,
	contract *common.Address,
	data []byte,
	amount sdk.Int,
	commit bool,
) (*evmtypes.MsgEthereumTxResponse, error) {
	nonce, err := k.accountKeeper.GetSequence(ctx, sender)
	if err != nil {
		return nil, err
	}

	from := common.BytesToAddress(sender.Bytes())

	msg := ethtypes.NewMessage(
		from,
		contract,
		nonce,
		amount.BigInt(),              // amount
		ctx.GasMeter().GasConsumed(), // gasLimit
		big.NewInt(0),                // gasPrice
		big.NewInt(0),                // gasFeeCap
		big.NewInt(0),                // gasTipCap
		data,
		ethtypes.AccessList{}, // AccessList
		!commit,               // isFake
	)

	res, err := k.evmKeeper.ApplyMessage(ctx, msg, evmtypes.NewNoOpTracer(), commit)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to apply ethereum core message")
	}

	if res.Failed() {
		return nil, sdkerrors.Wrap(evmtypes.ErrVMExecution, res.VmError)
	}

	return res, nil
}
