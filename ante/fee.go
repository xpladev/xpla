package ante

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	tmstrings "github.com/tendermint/tendermint/libs/strings"

	evmante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

const maxBypassMinFeeMsgGasUsage = uint64(200_000)

// MempoolFeeDecorator will check if the transaction's fee is at least as large
// as the local validator's minimum gasFee (defined in validator config).
//
// If fee is too low, decorator returns error and tx is rejected from mempool.
// Note this only applies when ctx.CheckTx = true. If fee is high enough or not
// CheckTx, then call next AnteHandler.
//
// CONTRACT: Tx must implement FeeTx to use MempoolFeeDecorator

type MempoolFeeDecorator struct {
	BypassMinFeeMsgTypes []string

	evmKeeper evmante.EVMKeeper
	feeKeeper evmtypes.FeeMarketKeeper
}

func NewMempoolFeeDecorator(bypassMsgTypes []string, fk evmtypes.FeeMarketKeeper, ek evmante.EVMKeeper) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		BypassMinFeeMsgTypes: bypassMsgTypes,
		feeKeeper:            fk,
		evmKeeper:            ek,
	}
}

func (mfd MempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	feeCoins := feeTx.GetFee()
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()

	// Only check for minimum fees if the execution mode is CheckTx and the tx does
	// not contain operator configured bypass messages. If the tx does contain
	// operator configured bypass messages only, it's total gas must be less than
	// or equal to a constant, otherwise minimum fees are checked to prevent spam.
	if ctx.IsCheckTx() && !simulate && !(mfd.bypassMinFeeMsgs(msgs) && gas <= uint64(len(msgs))*maxBypassMinFeeMsgGasUsage) {
		minGasPrice := mfd.feeKeeper.GetParams(ctx).MinGasPrice

		if !minGasPrice.IsZero() {

			evmParams := mfd.evmKeeper.GetParams(ctx)
			minGasPrices := sdk.DecCoins{
				{
					Denom:  evmParams.EvmDenom,
					Amount: minGasPrice,
				},
			}

			optionMinGasPrices := ctx.MinGasPrices()
			for _, coin := range optionMinGasPrices {
				if coin.Denom != "unom" {
					minGasPrices = minGasPrices.Add(
						sdk.DecCoin{
							Denom:  coin.Denom,
							Amount: coin.Amount,
						})
				}
			}

			requiredFees := make(sdk.Coins, 0)

			// Determine the required fees by multiplying each required minimum gas
			// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
			glDec := sdk.NewDec(int64(gas))
			for _, gp := range minGasPrices {
				fee := gp.Amount.Mul(glDec).Ceil().RoundInt()
				if fee.IsPositive() {
					requiredFees = requiredFees.Add(sdk.Coin{Denom: gp.Denom, Amount: fee})
				}
			}

			if !feeCoins.IsAnyGTE(requiredFees) {
				return ctx, sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "insufficient fees; got: %s required: %s", feeCoins, requiredFees)
			}
		}
	}

	return next(ctx, tx, simulate)
}

func (mfd MempoolFeeDecorator) bypassMinFeeMsgs(msgs []sdk.Msg) bool {
	for _, msg := range msgs {
		if tmstrings.StringInSlice(sdk.MsgTypeURL(msg), mfd.BypassMinFeeMsgTypes) {
			continue
		}

		return false
	}

	return true
}

type MinGasPriceDecorator struct {
	evmKeeper evmante.EVMKeeper
	feeKeeper evmtypes.FeeMarketKeeper
}

func NewMinGasPriceDecorator(fk evmtypes.FeeMarketKeeper, ek evmante.EVMKeeper) MinGasPriceDecorator {
	return MinGasPriceDecorator{feeKeeper: fk, evmKeeper: ek}
}

func (mpd MinGasPriceDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {

	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if simulate {
		return next(ctx, tx, simulate)
	}

	minGasPrice := mpd.feeKeeper.GetParams(ctx).MinGasPrice

	// short-circuit if min gas price is 0
	if minGasPrice.IsZero() {
		return next(ctx, tx, simulate)
	}

	feeCoins := feeTx.GetFee()
	if feeCoins.Len() == 0 {
		return next(ctx, tx, simulate)
	}

	evmParams := mpd.evmKeeper.GetParams(ctx)
	minGasPrices := sdk.DecCoins{
		{
			Denom:  evmParams.EvmDenom,
			Amount: minGasPrice,
		},
	}

	optionMinGasPrices := ctx.MinGasPrices()
	for _, coin := range optionMinGasPrices {
		if coin.Denom != "unom" {
			minGasPrices = minGasPrices.Add(
				sdk.DecCoin{
					Denom:  coin.Denom,
					Amount: coin.Amount,
				})
		}
	}

	gas := feeTx.GetGas()

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
