package keeper

import (
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	tmprotocrypto "github.com/tendermint/tendermint/proto/tendermint/crypto"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func (k Keeper) VolunteerValidatorCommissionProcess(ctx sdk.Context) error {
	volunteerValidators := k.GetVolunteerValidators(ctx)

	for strValAddr, _ := range volunteerValidators {
		valAddr, err := sdk.ValAddressFromBech32(strValAddr)
		if err != nil {
			return err
		}

		commissions, err := k.distKeeper.WithdrawValidatorCommission(ctx, valAddr)
		if err != nil {
			continue
		}

		err = k.distKeeper.FundCommunityPool(ctx, commissions, sdk.AccAddress(valAddr.Bytes()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) VolunteerValidatorUpdates(ctx sdk.Context) (updates []abci.ValidatorUpdate, err error) {
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	volunteerValidators := k.GetVolunteerValidators(ctx)

	for valAddr, volunteerValidator := range volunteerValidators {

		valAddress, validator, tmProtoPk, err := k.getValidatorInfo(ctx, valAddr)
		if err != nil {
			return nil, err
		}

		if volunteerValidator.IsDeleting { // unregister validator
			if err = k.deleteUnregisterVolunteerValidator(ctx, &updates, valAddress, validator, tmProtoPk); err != nil {
				return nil, err
			}
		} else if validator.IsJailed() { // jailed
			if err = k.updateJailedVolunteerValidator(ctx, &updates, volunteerValidator, valAddress, validator, tmProtoPk); err != nil {
				return nil, err
			}
		} else if !validator.IsBonded() { // if no unregister and jailed status, it must always be bonded
			if err = k.bondVolunteerValidator(ctx, volunteerValidator, valAddress, validator); err != nil {
				return nil, err
			}
		} else if volunteerValidator.Power == 0 {
			k.updateVolunteerValidatorPower(ctx, &updates, volunteerValidator, valAddress, validator, tmProtoPk, powerReduction)
		}

	}

	return updates, nil
}

func (k Keeper) deleteUnregisterVolunteerValidator(ctx sdk.Context, updates *[]abci.ValidatorUpdate, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey) error {

	// when the volunteer validator is not included in the active set
	if validator.IsBonded() && validator.Tokens.Equal(sdk.ZeroInt()) {
		if _, err := k.beginUnbondingValidator(ctx, validator); err != nil {
			return err
		}

		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  0,
		})
	}

	k.DeleteVolunteerValidator(ctx, valAddress)

	return nil
}

func (k Keeper) updateJailedVolunteerValidator(ctx sdk.Context, updates *[]abci.ValidatorUpdate, volunteerValidator types.VolunteerValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey) error {
	// when the volunteer validator is not included in the active set
	if validator.IsBonded() && volunteerValidator.Power != 0 {
		if _, err := k.beginUnbondingValidator(ctx, validator); err != nil {
			return err
		}

		volunteerValidator.Power = 0
		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  volunteerValidator.Power,
		})

		k.SetVolunteerValidator(ctx, valAddress, volunteerValidator)
	}

	return nil
}

func (k Keeper) bondVolunteerValidator(ctx sdk.Context, volunteerValidator types.VolunteerValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator) error {
	if _, err := k.bondValidator(ctx, validator); err != nil {
		return err
	}

	volunteerValidator.Power = 0
	k.SetVolunteerValidator(ctx, valAddress, volunteerValidator)

	return nil
}

func (k Keeper) updateVolunteerValidatorPower(ctx sdk.Context, updates *[]abci.ValidatorUpdate, volunteerValidator types.VolunteerValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey, powerReduction sdk.Int) {
	prevPower := k.stakingKeeper.GetLastValidatorPower(ctx, valAddress)
	newPower := validator.GetConsensusPower(powerReduction)

	if prevPower != newPower {
		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  newPower,
		})
	}

	volunteerValidator.Power = newPower
	k.SetVolunteerValidator(ctx, valAddress, volunteerValidator)
}
