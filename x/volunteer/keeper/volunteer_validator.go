package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func (k Keeper) GetVolunteerValidator(ctx context.Context, valAddress sdk.ValAddress) (types.VolunteerValidator, error) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.GetVolunteerValidatorKey(valAddress))
	if err != nil {
		return types.VolunteerValidator{}, err
	}

	volunteerValidator := types.VolunteerValidator{}
	k.cdc.MustUnmarshal(bz, &volunteerValidator)

	return volunteerValidator, nil
}

func (k Keeper) SetVolunteerValidator(ctx context.Context, valAddress sdk.ValAddress, validator types.VolunteerValidator) error {
	store := k.storeService.OpenKVStore(ctx)
	bz := k.cdc.MustMarshal(&validator)
	return store.Set(types.GetVolunteerValidatorKey(valAddress), bz)
}

func (k Keeper) DeleteVolunteerValidator(ctx context.Context, valAddress sdk.ValAddress) error {
	store := k.storeService.OpenKVStore(ctx)
	return store.Delete(types.GetVolunteerValidatorKey(valAddress))
}

func (k Keeper) GetVolunteerValidators(ctx context.Context) (volunteerValidators map[string]types.VolunteerValidator, err error) {
	volunteerValidators = make(map[string]types.VolunteerValidator)
	store := k.storeService.OpenKVStore(ctx)
	iterator, err := store.Iterator(types.VolunteerValidatorKey, storetypes.PrefixEndBytes(types.VolunteerValidatorKey))
	if err != nil {
		return nil, err
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		bz := iterator.Value()
		validator := types.VolunteerValidator{}
		k.cdc.MustUnmarshal(bz, &validator)

		volunteerValidators[validator.Address] = validator
	}

	return volunteerValidators, nil
}
