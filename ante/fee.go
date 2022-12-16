package ante

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	tmstrings "github.com/tendermint/tendermint/libs/strings"
	xplatypes "github.com/xpladev/xpla/types"

	xatpkeeper "github.com/xpladev/xpla/x/xatp/keeper"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
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
	ak                   authante.AccountKeeper
	xatpKeeper           xatpkeeper.Keeper
	smartQueryGasLimit   uint64
}

func NewMempoolFeeDecorator(bypassMsgTypes []string, ak authante.AccountKeeper, ck xatpkeeper.Keeper, sqgl uint64) MempoolFeeDecorator {
	return MempoolFeeDecorator{
		BypassMinFeeMsgTypes: bypassMsgTypes,
		ak:                   ak,
		xatpKeeper:           ck,
		smartQueryGasLimit:   sqgl,
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
		minGasPrices := ctx.MinGasPrices()

		var decimal int64
		var cw20Decimal *big.Int

		ctx = ctx.WithGasMeter(sdk.NewGasMeter(sdk.Gas(mfd.smartQueryGasLimit)))

		if !minGasPrices.IsZero() {
			var defaultGasPrice sdk.DecCoin
			for _, minGasPrice := range minGasPrices {
				if defaultGasPrice.Denom == xplatypes.DefaultDenom {
					defaultGasPrice = minGasPrice
					decimal = int64(len(defaultGasPrice.Amount.RoundInt().String()))
					cw20Decimal = new(big.Int).Exp(big.NewInt(10), big.NewInt(decimal), nil)
					break
				}
			}

			for _, fee := range feeCoins {
				xatp, found := mfd.xatpKeeper.GetXatp(ctx, fee.Denom)
				if found {
					ratioDec, err, _ := mfd.xatpKeeper.GetFeeInfoFromXATP(ctx, xatp.Token)
					if err != nil {
						return ctx, err
					}

					minGasPrices = minGasPrices.Add(
						sdk.DecCoin{
							Denom:  xatp.Denom,
							Amount: defaultGasPrice.Amount.Mul(ratioDec),
						})
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
					fee = sdk.NewDecFromBigInt(new(big.Int).Div(fee.Ceil().RoundInt().BigInt(), cw20Decimal))
				}

				requiredFees[i] = sdk.NewCoin(gp.Denom, fee.Ceil().RoundInt())
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

type DeductFeeDecorator struct {
	ak             authante.AccountKeeper
	bankKeeper     types.BankKeeper
	feegrantKeeper authante.FeegrantKeeper

	xatpKeeper         xatpkeeper.Keeper
	MinGasPrices       string
	smartQueryGasLimit uint64
}

func NewDeductFeeDecorator(ak authante.AccountKeeper, bk types.BankKeeper, fk authante.FeegrantKeeper, xk xatpkeeper.Keeper, minGasPrices string, sqgl uint64) DeductFeeDecorator {
	return DeductFeeDecorator{
		ak:             ak,
		bankKeeper:     bk,
		feegrantKeeper: fk,

		xatpKeeper:         xk,
		MinGasPrices:       minGasPrices,
		smartQueryGasLimit: sqgl,
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

	var isXpla bool = false
	for _, coin := range fee {
		denom := coin.Denom

		if denom == xplatypes.DefaultDenom {
			isXpla = true
		}
	}

	// deduct the fees
	if !feeTx.GetFee().IsZero() {

		if isXpla {
			err = DeductFees(dfd.bankKeeper, ctx, deductFeesFromAcc, feeTx.GetFee())
			if err != nil {
				return ctx, err
			}

		} else {

			ctx = ctx.WithGasMeter(sdk.NewGasMeter(sdk.Gas(dfd.smartQueryGasLimit)))
			for _, coin := range fee {

				denom := coin.Denom

				ratioDec, err, _ := dfd.xatpKeeper.GetFeeInfoFromXATP(ctx, denom)
				if err != nil {
					return ctx, err
				}

				var xplaDecimal sdk.Dec
				if denom != xplatypes.DefaultDenom {

					minGasPrices, err := sdk.ParseDecCoins(dfd.MinGasPrices)
					if err != nil {
						return ctx, err
					}

					var decimal int64
					for _, gp := range minGasPrices {

						if gp.Denom == xplatypes.DefaultDenom {
							decimal = int64(len(gp.Amount.RoundInt().String()))
						}
					}

					xplaDecimal = sdk.NewDecFromBigInt(new(big.Int).Mul(big.NewInt(1), new(big.Int).Exp(big.NewInt(10), big.NewInt(decimal), nil)))

					err = dfd.xatpKeeper.ExecuteContract(ctx, deductFeesFrom, denom, coin.Amount.String())
					if err != nil {
						return ctx, err
					}
				}

				xatpPayer := dfd.xatpKeeper.GetPayer(ctx)
				XatpPayerAcc, err := sdk.AccAddressFromBech32(xatpPayer)
				if err != nil {
					return ctx, err
				}

				accExists := dfd.ak.(authkeeper.AccountKeeper).HasAccount(ctx, XatpPayerAcc)
				if !accExists {
					return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "XATP payer account: %s not found", XatpPayerAcc)
				}

				deductFeesFromAcc0 := dfd.ak.GetAccount(ctx, XatpPayerAcc)

				xplaValue := sdk.NewDecFromInt(coin.Amount).Quo(ratioDec)
				xplaValue = xplaValue.Mul(xplaDecimal)
				xplaFee := sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, xplaValue.Ceil().RoundInt()))

				err = DeductFees(dfd.bankKeeper, ctx, deductFeesFromAcc0, xplaFee)
				if err != nil {
					return ctx, err
				}
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
