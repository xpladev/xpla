package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
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

	createValidatorMsg := p.ToCreateValidator()
	if err := k.CreateValidator(ctx, createValidatorMsg); err != nil {
		return err
	}

	k.SetZeroRewardValidator(ctx, valAddress, types.ZeroRewardValidator{
		Address:    valAddress.String(),
		Power:      0,
		IsDeleting: false,
	})
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
		if !validator.IsJailed() && zeroRewardValidator.Power != 0 {
			zeroRewardValidator.IsDeleting = true
			k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
		} else {
			k.DeleteZeroRewardValidator(ctx, valAddress)
		}
	}

	return nil
}
