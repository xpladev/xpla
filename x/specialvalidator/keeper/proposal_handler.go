package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func NewSpecialValidatorProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RegisterSpecialValidatorProposal:
			return handlerRegisterSpecialvalidatorProposal(ctx, k, c)
		case *types.UnregisterSpecialValidatorProposal:
			return handlerUnregisterSpecialvalidatorProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized special validator proposal content type: %T", c)
		}
	}
}

func handlerRegisterSpecialvalidatorProposal(ctx sdk.Context, k Keeper, p *types.RegisterSpecialValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	createValidatorMsg := p.ToCreateValidator()
	if err := k.CreateValidator(ctx, createValidatorMsg); err != nil {
		return err
	}

	k.SetSpecialValidator(ctx, valAddress, types.SpecialValidator{
		Address:    valAddress.String(),
		Power:      0,
		IsDeleting: false,
	})
	return nil
}

func handlerUnregisterSpecialvalidatorProposal(ctx sdk.Context, k Keeper, p *types.UnregisterSpecialValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	specialValidator, found := k.GetSpecialValidator(ctx, valAddress)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, `special validator (%s)`, valAddress.String())
	}

	if validator, found := k.stakingKeeper.GetValidator(ctx, valAddress); found {
		if !validator.IsJailed() && specialValidator.Power != 0 {
			specialValidator.IsDeleting = true
			k.SetSpecialValidator(ctx, valAddress, specialValidator)
		} else {
			k.DeleteSpecialValidator(ctx, valAddress)
		}
	}

	return nil
}
