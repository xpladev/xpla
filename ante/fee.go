package ante

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmstrings "github.com/tendermint/tendermint/libs/strings"

	ethermintante "github.com/evmos/ethermint/app/ante"
)

const maxBypassMinFeeMsgGasUsage = uint64(200_000)

// MinGasPriceDecorator will check if the transaction's fee is at least as large
// as the MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies for both CheckTx and DeliverTx
// If fee is high enough, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MinGasPriceDecorator
type MinGasPriceDecorator struct {
	BypassMinFeeMsgTypes []string

	feesKeeper ethermintante.FeeMarketKeeper
	evmKeeper  ethermintante.EVMKeeper
}

func NewMinGasPriceDecorator(fk ethermintante.FeeMarketKeeper, ek ethermintante.EVMKeeper, bypassMsgTypes []string) MinGasPriceDecorator {
	return MinGasPriceDecorator{feesKeeper: fk, evmKeeper: ek, BypassMinFeeMsgTypes: bypassMsgTypes}
}

func (mpd MinGasPriceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	minGasPrice := mpd.feesKeeper.GetParams(ctx).MinGasPrice
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()

	// Short-circuit if min gas price is 0 or if simulating
	if minGasPrice.IsZero() || simulate || (mpd.bypassMinFeeMsgs(msgs) && gas <= uint64(len(msgs))*maxBypassMinFeeMsgGasUsage) {
		return next(ctx, tx, simulate)
	}

	evmDenom := mpd.evmKeeper.GetParams(ctx).EvmDenom
	minGasPrices := sdk.DecCoins{
		{
			Denom:  evmDenom,
			Amount: minGasPrice,
		},
	}

	feeCoins := feeTx.GetFee()

	requiredFees := make(sdk.Coins, 0)

	// Determine the required fees by multiplying each required minimum gas
	// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
	gasLimit := sdk.NewDecFromBigInt(new(big.Int).SetUint64(gas))

	for _, gp := range minGasPrices {
		fee := gp.Amount.Mul(gasLimit).Ceil().RoundInt()
		if fee.IsPositive() {
			requiredFees = requiredFees.Add(sdk.Coin{Denom: gp.Denom, Amount: fee})
		}
	}

	if !feeCoins.IsAnyGTE(requiredFees) {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "provided fee < minimum global fee (%s < %s). Please increase the gas price.", feeCoins, requiredFees)
	}

	return next(ctx, tx, simulate)
}

func (mpd MinGasPriceDecorator) bypassMinFeeMsgs(msgs []sdk.Msg) bool {
	for _, msg := range msgs {
		if tmstrings.StringInSlice(sdk.MsgTypeURL(msg), mpd.BypassMinFeeMsgTypes) {
			continue
		}

		return false
	}

	return true
}
