package keeper

import (
	"context"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

func (k Keeper) mustGetValidator(ctx context.Context, addr sdk.ValAddress) types.Validator {
	validator, err := k.GetValidator(ctx, addr)
	if err != nil {
		panic(fmt.Sprintf("validator record not found for address: %X\n", addr))
	}

	return validator
}

// GetLastValidators gets the group of the bonded validators
func (k Keeper) GetLastValidators(ctx context.Context) (validators []types.Validator, err error) {
	store := k.storeService.OpenKVStore(ctx)

	// add the actual validator power sorted store
	maxValidators, err := k.MaxValidators(ctx)
	if err != nil {
		return nil, err
	}

	// add to volunteer validator count
	volunteerValidators, err := k.volunteerKeeper.GetVolunteerValidators(ctx)
	if err != nil {
		return nil, err
	}
	maxValidators += uint32(len(volunteerValidators))

	validators = make([]types.Validator, maxValidators)

	iterator, err := store.Iterator(types.LastValidatorPowerKey, storetypes.PrefixEndBytes(types.LastValidatorPowerKey))
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	i := 0
	for ; iterator.Valid(); iterator.Next() {
		// sanity check
		if i >= int(maxValidators) {
			panic("more validators than maxValidators found")
		}

		address := types.AddressFromLastValidatorPowerKey(iterator.Key())
		validator, err := k.GetValidator(ctx, address)
		if err != nil {
			return nil, err
		}

		validators[i] = validator
		i++
	}

	return validators[:i], nil // trim
}
