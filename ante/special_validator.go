package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	specialvalidatorkeeper "github.com/xpladev/xpla/x/specialvalidator/keeper"
)

type RejectDelegateSpecialValidatorDecorator struct {
	specialValidatorKeeper specialvalidatorkeeper.Keeper
}

func NewRejectDelegateSpecialValidatorDecorator(svk specialvalidatorkeeper.Keeper) RejectDelegateSpecialValidatorDecorator {
	return RejectDelegateSpecialValidatorDecorator{
		specialValidatorKeeper: svk,
	}
}

func (rdsvd RejectDelegateSpecialValidatorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {

	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgDelegate:
			if err := rdsvd.checkSpecialValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		case *stakingtypes.MsgBeginRedelegate:
			if err := rdsvd.checkSpecialValidator(ctx, msg.ValidatorSrcAddress); err != nil {
				return ctx, err
			}

			if err := rdsvd.checkSpecialValidator(ctx, msg.ValidatorDstAddress); err != nil {
				return ctx, err
			}

		case *stakingtypes.MsgUndelegate:
			if err := rdsvd.checkSpecialValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

func (rdsvd RejectDelegateSpecialValidatorDecorator) checkSpecialValidator(ctx sdk.Context, validatorAddress string) error {
	valAddress, err := sdk.ValAddressFromBech32(validatorAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if _, found := rdsvd.specialValidatorKeeper.GetSpecialValidator(ctx, valAddress); found {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "cannot delegate to special validator")
	}

	return nil
}
