package keeper

import (
	"fmt"

	tmstrings "github.com/tendermint/tendermint/libs/strings"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) CreateValidator(ctx sdk.Context, msg stakingtypes.MsgCreateValidator) error {
	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return err
	}

	// check to see if the pubkey or sender has been registered before
	if _, found := k.stakingKeeper.GetValidator(ctx, valAddr); found {
		return stakingtypes.ErrValidatorOwnerExists
	}

	pk, ok := msg.Pubkey.GetCachedValue().(cryptotypes.PubKey)
	if !ok {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidType, "Expecting cryptostakingtypes.PubKey, got %T", pk)
	}

	if _, found := k.stakingKeeper.GetValidatorByConsAddr(ctx, sdk.GetConsAddress(pk)); found {
		return stakingtypes.ErrValidatorPubKeyExists
	}

	bondDenom := k.stakingKeeper.BondDenom(ctx)
	if msg.Value.Denom != bondDenom {
		return sdkerrors.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", msg.Value.Denom, bondDenom,
		)
	}

	if _, err := msg.Description.EnsureLength(); err != nil {
		return err
	}

	cp := ctx.ConsensusParams()
	if cp != nil && cp.Validator != nil {
		if !tmstrings.StringInSlice(pk.Type(), cp.Validator.PubKeyTypes) {
			return sdkerrors.Wrapf(
				stakingtypes.ErrValidatorPubKeyTypeNotSupported,
				"got: %s, expected: %s", pk.Type(), cp.Validator.PubKeyTypes,
			)
		}
	}

	validator, err := stakingtypes.NewValidator(valAddr, pk, msg.Description)
	if err != nil {
		return err
	}
	commission := stakingtypes.NewCommissionWithTime(
		msg.Commission.Rate, msg.Commission.MaxRate,
		msg.Commission.MaxChangeRate, ctx.BlockHeader().Time,
	)

	validator, err = validator.SetInitialCommission(commission)
	if err != nil {
		return err
	}

	delegatorAddress, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	if err != nil {
		return err
	}

	validator.MinSelfDelegation = msg.MinSelfDelegation

	k.stakingKeeper.SetValidator(ctx, validator)
	k.stakingKeeper.SetValidatorByConsAddr(ctx, validator)
	k.stakingKeeper.SetNewValidatorByPowerIndex(ctx, validator)

	// call the after-creation hook
	k.stakingKeeper.AfterValidatorCreated(ctx, validator.GetOperator())

	// move coins from the msg.Address account to a (self-delegation) delegator account
	// the validator account and global shares are updated within here
	// NOTE source will always be from a wallet which are unbonded
	_, err = k.stakingKeeper.Delegate(ctx, delegatorAddress, msg.Value.Amount, stakingtypes.Unbonded, validator, true)
	if err != nil {
		return err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			stakingtypes.EventTypeCreateValidator,
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, msg.Value.String()),
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, stakingtypes.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.DelegatorAddress),
		),
	})

	return nil

}

// perform all the store operations for when a validator status becomes bonded
func (k Keeper) bondValidator(ctx sdk.Context, validator stakingtypes.Validator) (stakingtypes.Validator, error) {
	// delete the validator by power index, as the key will change
	k.stakingKeeper.DeleteValidatorByPowerIndex(ctx, validator)

	validator = validator.UpdateStatus(stakingtypes.Bonded)

	// save the now bonded validator record to the two referenced stores
	k.stakingKeeper.SetValidator(ctx, validator)
	k.stakingKeeper.SetValidatorByPowerIndex(ctx, validator)

	// delete from queue if present
	k.stakingKeeper.DeleteValidatorQueue(ctx, validator)

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return validator, err
	}
	k.stakingKeeper.AfterValidatorBonded(ctx, consAddr, validator.GetOperator())

	return validator, err
}

// perform all the store operations for when a validator begins unbonding
func (k Keeper) beginUnbondingValidator(ctx sdk.Context, validator stakingtypes.Validator) (stakingtypes.Validator, error) {
	params := k.stakingKeeper.GetParams(ctx)

	// delete the validator by power index, as the key will change
	k.stakingKeeper.DeleteValidatorByPowerIndex(ctx, validator)

	// sanity check
	if validator.Status != stakingtypes.Bonded {
		panic(fmt.Sprintf("should not already be unbonded or unbonding, validator: %v\n", validator))
	}

	validator = validator.UpdateStatus(stakingtypes.Unbonding)

	// set the unbonding completion time and completion height appropriately
	validator.UnbondingTime = ctx.BlockHeader().Time.Add(params.UnbondingTime)
	validator.UnbondingHeight = ctx.BlockHeader().Height

	// save the now unbonded validator record and power index
	k.stakingKeeper.SetValidator(ctx, validator)
	k.stakingKeeper.SetValidatorByPowerIndex(ctx, validator)

	// Adds to unbonding validator queue
	k.stakingKeeper.InsertUnbondingValidatorQueue(ctx, validator)

	// trigger hook
	consAddr, err := validator.GetConsAddr()
	if err != nil {
		return validator, err
	}
	k.stakingKeeper.AfterValidatorBeginUnbonding(ctx, consAddr, validator.GetOperator())

	return validator, nil
}
