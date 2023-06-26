package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
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
