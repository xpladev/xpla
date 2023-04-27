package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func (k Keeper) GetSpecialValidator(ctx sdk.Context, valAddress sdk.ValAddress) (types.SpecialValidator, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetSpecialValidatorKey(valAddress))
	if bz == nil {
		return types.SpecialValidator{}, false
	}

	specialValidator := types.SpecialValidator{}
	k.cdc.MustUnmarshal(bz, &specialValidator)

	return specialValidator, true
}

func (k Keeper) SetSpecialValidator(ctx sdk.Context, valAddress sdk.ValAddress, validator types.SpecialValidator) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&validator)
	store.Set(types.GetSpecialValidatorKey(valAddress), bz)
}

func (k Keeper) DeleteSpecialValidator(ctx sdk.Context, valAddress sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetSpecialValidatorKey(valAddress))
}

func (k Keeper) GetSpecialValidators(ctx sdk.Context) (specialValidators map[string]types.SpecialValidator) {
	specialValidators = make(map[string]types.SpecialValidator)
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.SpecialValidatorKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valAddress := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iterator.Key()))
		bz := iterator.Value()
		validator := types.SpecialValidator{}
		k.cdc.MustUnmarshal(bz, &validator)

		specialValidators[valAddress.String()] = validator
	}

	return specialValidators
}
