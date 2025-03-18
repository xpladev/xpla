package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(k Keeper) types.MsgServer {
	return &msgServer{Keeper: k}
}

func (k msgServer) RegisterVolunteerValidator(goCtx context.Context, req *types.MsgRegisterVolunteerValidator) (*types.MsgRegisterVolunteerValidatorResponse, error) {
	if k.authority != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	valAddress, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	if _, err = k.stakingKeeper.GetValidator(ctx, valAddress); err == nil {
		return nil, stakingtypes.ErrValidatorOwnerExists
	}

	k.SetVolunteerValidator(ctx, valAddress, types.NewVolunteerValidator(valAddress, 0))

	createValidatorMsg := req.ToCreateValidator()
	if err := k.CreateValidator(ctx, createValidatorMsg); err != nil {
		return nil, err
	}

	return &types.MsgRegisterVolunteerValidatorResponse{}, nil
}

func (k msgServer) UnregisterVolunteerValidator(goCtx context.Context, req *types.MsgUnregisterVolunteerValidator) (*types.MsgUnregisterVolunteerValidatorResponse, error) {
	if k.authority != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	valAddress, err := sdk.ValAddressFromBech32(req.ValidatorAddress)
	if err != nil {
		return nil, err
	}

	_, err = k.GetVolunteerValidator(ctx, valAddress)
	if err != nil {
		return nil, errorsmod.Wrapf(err, `volunteer validator (%s)`, valAddress.String())
	}

	if validator, err := k.stakingKeeper.GetValidator(ctx, valAddress); err == nil {
		_, _, err := k.stakingKeeper.Undelegate(ctx, sdk.AccAddress(valAddress), valAddress, validator.DelegatorShares)
		if err != nil {
			return nil, err
		}

		k.DeleteVolunteerValidator(ctx, valAddress)
	}

	return &types.MsgUnregisterVolunteerValidatorResponse{}, nil
}
