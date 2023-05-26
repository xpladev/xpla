package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func (k Keeper) GetVolunteerValidator(ctx sdk.Context, valAddress sdk.ValAddress) (types.VolunteerValidator, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetVolunteerValidatorKey(valAddress))
	if bz == nil {
		return types.VolunteerValidator{}, false
	}

	volunteerValidator := types.VolunteerValidator{}
	k.cdc.MustUnmarshal(bz, &volunteerValidator)

	return volunteerValidator, true
}

func (k Keeper) SetVolunteerValidator(ctx sdk.Context, valAddress sdk.ValAddress, validator types.VolunteerValidator) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&validator)
	store.Set(types.GetVolunteerValidatorKey(valAddress), bz)
}

func (k Keeper) DeleteVolunteerValidator(ctx sdk.Context, valAddress sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetVolunteerValidatorKey(valAddress))
}

func (k Keeper) GetVolunteerValidators(ctx sdk.Context) (volunteerValidators map[string]types.VolunteerValidator) {
	volunteerValidators = make(map[string]types.VolunteerValidator)
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.VolunteerValidatorKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valAddress := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iterator.Key()))
		bz := iterator.Value()
		validator := types.VolunteerValidator{}
		k.cdc.MustUnmarshal(bz, &validator)

		volunteerValidators[valAddress.String()] = validator
	}

	return volunteerValidators
}
