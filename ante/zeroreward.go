package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	zerorewardkeeper "github.com/xpladev/xpla/x/zeroreward/keeper"
)

type RejectDelegateZeroRewardValidatorDecorator struct {
	zeroRewardKeeper zerorewardkeeper.Keeper
}

func NewRejectDelegateZeroRewardValidatorDecorator(svk zerorewardkeeper.Keeper) RejectDelegateZeroRewardValidatorDecorator {
	return RejectDelegateZeroRewardValidatorDecorator{
		zeroRewardKeeper: svk,
	}
}

func (rdsvd RejectDelegateZeroRewardValidatorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Context, err error) {

	for _, msg := range tx.GetMsgs() {
		switch msg := msg.(type) {
		case *stakingtypes.MsgDelegate:
			if err := rdsvd.checkZeroRewardValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		case *stakingtypes.MsgBeginRedelegate:
			if err := rdsvd.checkZeroRewardValidator(ctx, msg.ValidatorSrcAddress); err != nil {
				return ctx, err
			}

			if err := rdsvd.checkZeroRewardValidator(ctx, msg.ValidatorDstAddress); err != nil {
				return ctx, err
			}

		case *stakingtypes.MsgUndelegate:
			if err := rdsvd.checkZeroRewardValidator(ctx, msg.ValidatorAddress); err != nil {
				return ctx, err
			}
		}
	}

	return next(ctx, tx, simulate)
}

func (rdsvd RejectDelegateZeroRewardValidatorDecorator) checkZeroRewardValidator(ctx sdk.Context, validatorAddress string) error {
	valAddress, err := sdk.ValAddressFromBech32(validatorAddress)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
	}
	if _, found := rdsvd.zeroRewardKeeper.GetZeroRewardValidator(ctx, valAddress); found {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "cannot delegate to zero reward validator")
	}

	return nil
}
