package ante

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	tmstrings "github.com/tendermint/tendermint/libs/strings"

	ethermintante "github.com/evmos/ethermint/app/ante"
	xplatypes "github.com/xpladev/xpla/types"

	xatpkeeper "github.com/xpladev/xpla/x/xatp/keeper"
)

const maxBypassMinFeeMsgGasUsage = uint64(200_000)

// MinGasPriceDecorator will check if the transaction's fee is at least as large
// as the MinGasPrices param. If fee is too low, decorator returns error and tx
// is rejected. This applies for both CheckTx and DeliverTx
// If fee is high enough, then call next AnteHandler
// CONTRACT: Tx must implement FeeTx to use MinGasPriceDecorator
type MempoolFeeDecorator struct {
	BypassMinFeeMsgTypes []string

	ak         authante.AccountKeeper
	xatpKeeper xatpkeeper.Keeper
	feesKeeper ethermintante.FeeMarketKeeper
	evmKeeper  ethermintante.EVMKeeper
}

func NewMempoolFeeDecorator(bypassMsgTypes []string, ak authante.AccountKeeper, ck xatpkeeper.Keeper, fk ethermintante.FeeMarketKeeper, ek ethermintante.EVMKeeper) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		BypassMinFeeMsgTypes: bypassMsgTypes,
		ak:                   ak,
		xatpKeeper:           ck,
		feesKeeper:           fk,
		evmKeeper:            ek,
	}
}

func (mfd MempoolFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	minGasPrice := mfd.feesKeeper.GetParams(ctx).MinGasPrice
	gas := feeTx.GetGas()
	msgs := feeTx.GetMsgs()
	feeCoins := feeTx.GetFee()

	evmDenom := mfd.evmKeeper.GetParams(ctx).EvmDenom
	minGasPrices := sdk.DecCoins{
		{
			Denom:  evmDenom,
			Amount: minGasPrice,
		},
	}
	// Only check for minimum fees if the execution mode is CheckTx and the tx does
	// not contain operator configured bypass messages. If the tx does contain
	// operator configured bypass messages only, it's total gas must be less than
	// or equal to a constant, otherwise minimum fees are checked to prevent spam.
	if ctx.IsCheckTx() && !simulate && !(mfd.bypassMinFeeMsgs(msgs) && gas <= uint64(len(msgs))*maxBypassMinFeeMsgGasUsage) {
		if !minGasPrices.IsZero() {
			var defaultGasPrice sdk.DecCoin
			for _, minGasPrice := range minGasPrices {
				if minGasPrice.Denom == xplatypes.DefaultDenom {
					defaultGasPrice = minGasPrice
					break
				}
			}

			for _, fee := range feeCoins {
				xatp, found := mfd.xatpKeeper.GetXatp(ctx, fee.Denom)
				if found {
					ratioDec, err := mfd.xatpKeeper.GetFeeInfoFromXATP(ctx, xatp.Denom)
					if err != nil {
						return ctx, err
					}

					minGasPrices = minGasPrices.Add(sdk.NewDecCoinFromDec(xatp.Denom, defaultGasPrice.Amount.Mul(ratioDec)))
				}
			}

			requiredFees := make(sdk.Coins, len(minGasPrices))

			// Determine the required fees by multiplying each required minimum gas
			// price by the gas limit, where fee = ceil(minGasPrice * gasLimit).
			glDec := sdk.NewDec(int64(gas))
			var fee sdk.Dec
			for i, gp := range minGasPrices {
				fee = gp.Amount.Mul(glDec)

				if gp.Denom != xplatypes.DefaultDenom {
					xatp, found := mfd.xatpKeeper.GetXatp(ctx, gp.Denom)
					if found {
						decimal := sdk.NewDecFromIntWithPrec(sdk.NewInt(10), int64(xatp.Decimals))
						fee = fee.Ceil().Quo(decimal)
					}
				}

				requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
			}

			feeCoins := feeTx.GetFee()

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

type DeductFeeDecorator struct {
	ak             authante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper authante.FeegrantKeeper

	xatpKeeper xatpkeeper.Keeper
}

func NewDeductFeeDecorator(ak authante.AccountKeeper, bk types.BankKeeper, fk authante.FeegrantKeeper, xk xatpkeeper.Keeper) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:             ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,

		xatpKeeper: xk,
	}
}

func (dfd DeductFeeDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	feeTx, ok := tx.(sdk.FeeTx)
	if !ok {
		return ctx, sdkerrors.Wrap(sdkerrors.ErrTxDecode, "Tx must be a FeeTx")
	}

	if addr := dfd.ak.GetModuleAddress(types.FeeCollectorName); addr == nil {
		return ctx, fmt.Errorf("Fee collector module account (%s) has not been set", types.FeeCollectorName)
	}

	fee := feeTx.GetFee()
	feePayer := feeTx.FeePayer()
	feeGranter := feeTx.FeeGranter()

	deductFeesFrom := feePayer

	// if feegranter set deduct fee from feegranter account.
	// this works with only when feegrant enabled.
	if feeGranter != nil {
		if dfd.feegrantKeeper == nil {
			return ctx, sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "fee grants are not enabled")
		} else if !feeGranter.Equals(feePayer) {
			err := dfd.feegrantKeeper.UseGrantedFees(ctx, feeGranter, feePayer, fee, tx.GetMsgs())

			if err != nil {
				return ctx, sdkerrors.Wrapf(err, "%s not allowed to pay fees from %s", feeGranter, feePayer)
			}
		}

		deductFeesFrom = feeGranter
	}

	deductFeesFromAcc := dfd.ak.GetAccount(ctx, deductFeesFrom)
	if deductFeesFromAcc == nil {
		return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "fee payer address: %s does not exist", deductFeesFrom)
	}

	// deduct the fees
	if !feeTx.GetFee().IsZero() {
		nativeFees := sdk.Coins{}
		xatpFees := sdk.Coins{}

		for _, fee := range feeTx.GetFee() {
			xatp, found := dfd.xatpKeeper.GetXatp(ctx, fee.Denom)
			if !found {
				nativeFees = nativeFees.Add(fee)
				continue
			}

			err := dfd.xatpKeeper.PayXATP(ctx, deductFeesFrom, xatp.Denom, fee.Amount.String())
			if err != nil {
				return ctx, err
			}

			ratioDec, err := dfd.xatpKeeper.GetFeeInfoFromXATP(ctx, xatp.Denom)
			if err != nil {
				return ctx, err
			}

			feeAmount := sdk.NewDecFromIntWithPrec(fee.Amount, int64(xatp.Decimals))
			defaultFeeAmountDec := feeAmount.Quo(ratioDec)
			defaultFeeAmount := defaultFeeAmountDec.Mul(sdk.NewDecWithPrec(10, xplatypes.DefaultDenomPrecision)).Ceil().BigInt()
			xatpFees = xatpFees.Add(sdk.NewCoin(xplatypes.DefaultDenom, sdk.NewIntFromBigInt(defaultFeeAmount)))
		}

		if !nativeFees.Empty() {
			err = DeductFees(dfd.bankKeeper, ctx, deductFeesFromAcc, nativeFees)
			if err != nil {
				return ctx, err
			}
		}

		if !xatpFees.Empty() {
			xatpPayer := dfd.xatpKeeper.GetPayer(ctx)
			xatpPayerAcc, err := sdk.AccAddressFromBech32(xatpPayer)
			if err != nil {
				return ctx, err
			}

			deductFeeAccount := dfd.ak.GetAccount(ctx, xatpPayerAcc)
			err = DeductFees(dfd.bankKeeper, ctx, deductFeeAccount, xatpFees)
			if err != nil {
				return ctx, err
			}
		}
	}

	events := sdk.Events{sdk.NewEvent(sdk.EventTypeTx,
		sdk.NewAttribute(sdk.AttributeKeyFee, feeTx.GetFee().String()),
	)}

	ctx.EventManager().EmitEvents(events)

	return next(ctx, tx, simulate)
}

func DeductFees(bankKeeper types.BankKeeper, ctx sdk.Context, acc types.AccountI, fees sdk.Coins) error {
	if !fees.IsValid() {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFee, "invalid fee amount: %s", fees)
	}

	err := bankKeeper.SendCoinsFromAccountToModule(ctx, acc.GetAddress(), types.FeeCollectorName, fees)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInsufficientFunds, err.Error())
	}

	return nil
}
