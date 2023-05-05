package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) AfterValidatorBonded(ctx sdk.Context, valAddress sdk.ValAddress) {
	zeroRewardValidator, found := k.GetZeroRewardValidator(ctx, valAddress)

	if found {
		power := k.stakingKeeper.GetLastValidatorPower(ctx, valAddress)
		zeroRewardValidator.Power = power
		k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
	}
}

func (k Keeper) AfterValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	_, found := k.GetZeroRewardValidator(ctx, valAddr)

	if found {
		val, found := k.stakingKeeper.GetValidator(ctx, valAddr)
		if !found {
			panic(fmt.Errorf(`not found validator (%s)`, val.String()))
		}

		if !val.IsJailed() {
			val = val.UpdateStatus(stakingtypes.Bonded)

			k.stakingKeeper.SetValidator(ctx, val)
			k.stakingKeeper.DeleteValidatorQueue(ctx, val)
			k.stakingKeeper.AfterValidatorCreated(ctx, valAddr)
			k.stakingKeeper.AfterValidatorBonded(ctx, consAddr, valAddr)
		}
	}
}

func (k Keeper) AfterDelegationModified(ctx sdk.Context, valAddress sdk.ValAddress) {
	zeroRewardValidator, found := k.GetZeroRewardValidator(ctx, valAddress)
	if found {
		power := k.stakingKeeper.GetLastValidatorPower(ctx, valAddress)
		if power > 0 {
			zeroRewardValidator.Power = power
			k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
		}
	}
}

// Hooks wrapper struct for zeroreward keeper
type Hooks struct {
	k Keeper
}

var _ stakingtypes.StakingHooks = Hooks{}

// Return the wrapper struct
func (k Keeper) Hooks() Hooks {
	return Hooks{k}
}

// Implements sdk.ValidatorHooks
func (h Hooks) AfterValidatorBonded(ctx sdk.Context, _ sdk.ConsAddress, valAddr sdk.ValAddress) {
	h.k.AfterValidatorBonded(ctx, valAddr)
}

// Implements sdk.ValidatorHooks
func (h Hooks) AfterValidatorRemoved(ctx sdk.Context, consAddr sdk.ConsAddress, valAddr sdk.ValAddress) {
	h.k.AfterValidatorRemoved(ctx, consAddr, valAddr)
}

// Implements sdk.ValidatorHooks
func (h Hooks) AfterDelegationModified(ctx sdk.Context, _ sdk.AccAddress, valAddr sdk.ValAddress) {
	h.k.AfterDelegationModified(ctx, valAddr)
}

func (h Hooks) AfterValidatorCreated(_ sdk.Context, _ sdk.ValAddress)                            {}
func (h Hooks) AfterValidatorBeginUnbonding(_ sdk.Context, _ sdk.ConsAddress, _ sdk.ValAddress)  {}
func (h Hooks) BeforeValidatorModified(_ sdk.Context, _ sdk.ValAddress)                          {}
func (h Hooks) BeforeDelegationCreated(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress)        {}
func (h Hooks) BeforeDelegationSharesModified(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress) {}
func (h Hooks) BeforeDelegationRemoved(_ sdk.Context, _ sdk.AccAddress, _ sdk.ValAddress)        {}
func (h Hooks) BeforeValidatorSlashed(_ sdk.Context, _ sdk.ValAddress, _ sdk.Dec)                {}
