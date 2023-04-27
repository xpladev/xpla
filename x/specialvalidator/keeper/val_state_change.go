package keeper

import (
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k Keeper) SpecialValidatorCommissionProcess(ctx sdk.Context) error {
	specialValidators := k.GetSpecialValidators(ctx)

	for strValAddr, _ := range specialValidators {
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

func (k Keeper) SpecialValidatorUpdates(ctx sdk.Context) (updates []abci.ValidatorUpdate, err error) {
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	specialValidators := k.GetSpecialValidators(ctx)

	for valAddr, specialValidator := range specialValidators {
		addr, err := sdk.ValAddressFromBech32(valAddr)
		if err != nil {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "validator address (%s)", valAddr)
		}

		power := k.stakingKeeper.GetLastValidatorPower(ctx, addr)

		if power > 0 {
			continue
		}

		validator, found := k.stakingKeeper.GetValidator(ctx, addr)
		if !found {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "validator (%s)", addr.String())
		}

		tmProtoPk, err := validator.TmConsPublicKey()
		if err != nil {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrInvalidPubKey, "validator (%s)", addr.String())
		}

		// unregister validator
		if specialValidator.IsDeleting {
			_, err = k.beginUnbondingValidator(ctx, validator)
			if err != nil {
				return nil, err
			}

			updates = append(updates, abci.ValidatorUpdate{
				PubKey: tmProtoPk,
				Power:  0,
			})
			k.DeleteSpecialValidator(ctx, addr)
			continue
		}

		// jailed
		if validator.IsJailed() {

			// when only special validator, must be unbonding
			if specialValidator.Power != 0 {
				_, err = k.beginUnbondingValidator(ctx, validator)
				if err != nil {
					return nil, err
				}

				specialValidator.Power = 0
				updates = append(updates, abci.ValidatorUpdate{
					PubKey: tmProtoPk,
					Power:  specialValidator.Power,
				})

				k.SetSpecialValidator(ctx, addr, specialValidator)
			}

			continue
		}

		newPower := validator.PotentialConsensusPower(powerReduction)
		if specialValidator.Power != newPower {
			if specialValidator.Power == 0 {
				k.bondValidator(ctx, validator)
			}

			updates = append(updates, abci.ValidatorUpdate{
				PubKey: tmProtoPk,
				Power:  newPower,
			})

			specialValidator.Power = newPower
			k.SetSpecialValidator(ctx, addr, specialValidator)
		}

	}

	return updates, nil
}
