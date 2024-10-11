package ante

import (
	errorsmod "cosmossdk.io/errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type RejectDelegateVolunteerValidatorDecorator struct {
	volunteerKeeper VolunteerKeeper
}

func NewRejectDelegateVolunteerValidatorDecorator(vk VolunteerKeeper) RejectDelegateVolunteerValidatorDecorator {
	return RejectDelegateVolunteerValidatorDecorator{
		volunteerKeeper: vk,
	}
}

func (rdvvd RejectDelegateVolunteerValidatorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {
	if ctx.GasMeter().Limit() == 0 && simulate {
		return next(ctx, tx, simulate)
	}

	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgDelegate:
			if err := rdvvd.checkVolunteerValidator(ctx, msg.ValidatorAddress, msg.DelegatorAddress); err != nil {
				return ctx, err
			}
		case *stakingtypes.MsgBeginRedelegate:
			if err := rdvvd.checkVolunteerValidator(ctx, msg.ValidatorSrcAddress, msg.DelegatorAddress); err != nil {
				return ctx, err
			}

			if err := rdvvd.checkVolunteerValidator(ctx, msg.ValidatorDstAddress, msg.DelegatorAddress); err != nil {
				return ctx, err
			}

		case *stakingtypes.MsgUndelegate:
			if err := rdvvd.checkVolunteerValidator(ctx, msg.ValidatorAddress, msg.DelegatorAddress); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

func (rdvvd RejectDelegateVolunteerValidatorDecorator) checkVolunteerValidator(ctx sdk.Context, validatorAddress, delegatorAddress string) error {
	valAddress, err := sdk.ValAddressFromBech32(validatorAddress)
	if err != nil {
		return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
	}

	if _, err := rdvvd.volunteerKeeper.GetVolunteerValidator(ctx, valAddress); err == nil {
		delAddress, err := sdk.AccAddressFromBech32(delegatorAddress)
		if err != nil {
			return errorsmod.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
		}

		if delAddress.Equals(valAddress) {
			return nil
		}

		return errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "cannot delegate to volunteer validator")
	}

	return nil
}
