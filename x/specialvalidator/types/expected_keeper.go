package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetParams(ctx sdk.Context) stakingtypes.Params
	DeleteValidatorByPowerIndex(ctx sdk.Context, validator stakingtypes.Validator)
	SetValidatorByPowerIndex(ctx sdk.Context, validator stakingtypes.Validator)
	InsertUnbondingValidatorQueue(ctx sdk.Context, val stakingtypes.Validator)
	GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool)
	GetValidatorByConsAddr(ctx sdk.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, found bool)
	BondDenom(ctx sdk.Context) (res string)
	SetValidator(ctx sdk.Context, validator stakingtypes.Validator)
	SetValidatorByConsAddr(ctx sdk.Context, validator stakingtypes.Validator) error
	SetNewValidatorByPowerIndex(ctx sdk.Context, validator stakingtypes.Validator)
	AfterValidatorCreated(ctx sdk.Context, valAddr sdk.ValAddress)
	AfterValidatorBonded(ctx sdk.Context, address sdk.ConsAddress, _ sdk.ValAddress)
	AfterValidatorBeginUnbonding(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress)
	Delegate(
		ctx sdk.Context, delAddr sdk.AccAddress, bondAmt sdk.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool,
	) (newShares sdk.Dec, err error)
	DeleteValidatorQueue(ctx sdk.Context, val stakingtypes.Validator)
	GetLastValidatorPower(ctx sdk.Context, operator sdk.ValAddress) (power int64)
	PowerReduction(ctx sdk.Context) sdk.Int
}
