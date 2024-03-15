package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func NewVolunteerValidatorProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RegisterVolunteerValidatorProposal:
			return handlerRegisterVolunteerValidatorProposal(ctx, k, c)
		case *types.UnregisterVolunteerValidatorProposal:
			return handlerUnregisterVolunteerValidatorProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized volunteer validator proposal content type: %T", c)
		}
	}
}

func handlerRegisterVolunteerValidatorProposal(ctx sdk.Context, k Keeper, p *types.RegisterVolunteerValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	if _, found := k.stakingKeeper.GetValidator(ctx, valAddress); found {
		return stakingtypes.ErrValidatorOwnerExists
	}

	k.SetVolunteerValidator(ctx, valAddress, types.NewVolunteerValidator(valAddress, 0))

	createValidatorMsg := p.ToCreateValidator()
	if err := k.CreateValidator(ctx, createValidatorMsg); err != nil {
		return err
	}

	return nil
}

func handlerUnregisterVolunteerValidatorProposal(ctx sdk.Context, k Keeper, p *types.UnregisterVolunteerValidatorProposal) error {
	valAddress, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	_, found := k.GetVolunteerValidator(ctx, valAddress)
	if !found {
		return sdkerrors.Wrapf(sdkerrors.ErrNotFound, `volunteer validator (%s)`, valAddress.String())
	}

	if validator, found := k.stakingKeeper.GetValidator(ctx, valAddress); found {
		_, err := k.stakingKeeper.Undelegate(ctx, sdk.AccAddress(valAddress), valAddress, validator.DelegatorShares)
		if err != nil {
			return err
		}

		k.DeleteVolunteerValidator(ctx, valAddress)
	}

	return nil
}
