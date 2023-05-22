package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func NewZeroRewardValidatorProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RegisterZeroRewardValidatorProposal:
			return handlerRegisterZeroRewardValidatorProposal(ctx, k, c)
		case *types.UnregisterZeroRewardValidatorProposal:
			return handlerUnregisterZeroRewardValidatorProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized zero reward validator proposal content type: %T", c)
		}
	}
}

func handlerRegisterZeroRewardValidatorProposal(ctx sdk.Context, k Keeper, p *types.RegisterZeroRewardValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	if _, found := k.stakingKeeper.GetValidator(ctx, valAddress); found {
		return stakingtypes.ErrValidatorOwnerExists
	}

	k.SetZeroRewardValidator(ctx, valAddress, types.NewZeroRewardValidator(valAddress, 0))

	createValidatorMsg := p.ToCreateValidator()
	if err := k.CreateValidator(ctx, createValidatorMsg); err != nil {
		return err
	}

	return nil
}

func handlerUnregisterZeroRewardValidatorProposal(ctx sdk.Context, k Keeper, p *types.UnregisterZeroRewardValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	zeroRewardValidator, found := k.GetZeroRewardValidator(ctx, valAddress)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, `zero reward validator (%s)`, valAddress.String())
	}

	if validator, found := k.stakingKeeper.GetValidator(ctx, valAddress); found {
		_, err := k.stakingKeeper.Undelegate(ctx, sdk.AccAddress(valAddress), valAddress, validator.DelegatorShares)
		if err != nil {
			return err
		}

		if !validator.IsJailed() && zeroRewardValidator.Power != 0 {
			zeroRewardValidator.Delete()
			k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
		} else {
			k.DeleteZeroRewardValidator(ctx, valAddress)
		}
	}

	return nil
}
