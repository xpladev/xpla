package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	volunteerkeeper "github.com/xpladev/xpla/x/volunteer/keeper"
)

type RejectDelegateVolunteerValidatorDecorator struct {
	volunteerKeeper volunteerkeeper.Keeper
}

func NewRejectDelegateVolunteerValidatorDecorator(svk volunteerkeeper.Keeper) RejectDelegateVolunteerValidatorDecorator {
	return RejectDelegateVolunteerValidatorDecorator{
		volunteerKeeper: svk,
	}
}

func (rdsvd RejectDelegateVolunteerValidatorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {

	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgDelegate:
			if err := rdsvd.checkVolunteerValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		case *stakingtypes.MsgBeginRedelegate:
			if err := rdsvd.checkVolunteerValidator(ctx, msg.ValidatorSrcAddress); err != nil {
				return ctx, err
			}

			if err := rdsvd.checkVolunteerValidator(ctx, msg.ValidatorDstAddress); err != nil {
				return ctx, err
			}

		case *stakingtypes.MsgUndelegate:
			if err := rdsvd.checkVolunteerValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

func (rdsvd RejectDelegateVolunteerValidatorDecorator) checkVolunteerValidator(ctx sdk.Context, validatorAddress string) error {
	valAddress, err := sdk.ValAddressFromBech32(validatorAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if _, found := rdsvd.volunteerKeeper.GetVolunteerValidator(ctx, valAddress); found {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "cannot delegate to volunteer validator")
	}

	return nil
}
