package keeper

import (
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k Keeper) ZeroRewardValidatorCommissionProcess(ctx sdk.Context) error {
	zeroRewardValidators := k.GetZeroRewardValidators(ctx)

	for strValAddr, _ := range zeroRewardValidators {
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

func (k Keeper) ZeroRewardValidatorUpdates(ctx sdk.Context) (updates []abci.ValidatorUpdate, err error) {
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	zeroRewardValidators := k.GetZeroRewardValidators(ctx)

	for valAddr, zeroRewardValidator := range zeroRewardValidators {
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
		if zeroRewardValidator.IsDeleting {
			_, err = k.beginUnbondingValidator(ctx, validator)
			if err != nil {
				return nil, err
			}

			updates = append(updates, abci.ValidatorUpdate{
				PubKey: tmProtoPk,
				Power:  0,
			})
			k.DeleteZeroRewardValidator(ctx, addr)
			continue
		}

		// jailed
		if validator.IsJailed() {

			// when only zero reward validator, must be unbonding
			if zeroRewardValidator.Power != 0 {
				_, err = k.beginUnbondingValidator(ctx, validator)
				if err != nil {
					return nil, err
				}

				zeroRewardValidator.Power = 0
				updates = append(updates, abci.ValidatorUpdate{
					PubKey: tmProtoPk,
					Power:  zeroRewardValidator.Power,
				})

				k.SetZeroRewardValidator(ctx, addr, zeroRewardValidator)
			}

			continue
		}

		newPower := validator.PotentialConsensusPower(powerReduction)
		if zeroRewardValidator.Power != newPower {
			if zeroRewardValidator.Power == 0 {
				k.bondValidator(ctx, validator)
			}

			updates = append(updates, abci.ValidatorUpdate{
				PubKey: tmProtoPk,
				Power:  newPower,
			})

			zeroRewardValidator.Power = newPower
			k.SetZeroRewardValidator(ctx, addr, zeroRewardValidator)
		}

	}

	return updates, nil
}
