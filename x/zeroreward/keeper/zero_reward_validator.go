package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func (k Keeper) GetZeroRewardValidator(ctx sdk.Context, valAddress sdk.ValAddress) (types.ZeroRewardValidator, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GetZeroRewardValidatorKey(valAddress))
	if bz == nil {
		return types.ZeroRewardValidator{}, false
	}

	zeroRewardValidator := types.ZeroRewardValidator{}
	k.cdc.MustUnmarshal(bz, &zeroRewardValidator)

	return zeroRewardValidator, true
}

func (k Keeper) SetZeroRewardValidator(ctx sdk.Context, valAddress sdk.ValAddress, validator types.ZeroRewardValidator) {
	store := ctx.KVStore(k.storeKey)
	bz := k.cdc.MustMarshal(&validator)
	store.Set(types.GetZeroRewardValidatorKey(valAddress), bz)
}

func (k Keeper) DeleteZeroRewardValidator(ctx sdk.Context, valAddress sdk.ValAddress) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.GetZeroRewardValidatorKey(valAddress))
}

func (k Keeper) GetZeroRewardValidators(ctx sdk.Context) (zeroRewardValidators map[string]types.ZeroRewardValidator) {
	zeroRewardValidators = make(map[string]types.ZeroRewardValidator)
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.ZeroRewardValidatorKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		valAddress := sdk.ValAddress(stakingtypes.AddressFromValidatorsKey(iterator.Key()))
		bz := iterator.Value()
		validator := types.ZeroRewardValidator{}
		k.cdc.MustUnmarshal(bz, &validator)

		zeroRewardValidators[valAddress.String()] = validator
	}

	return zeroRewardValidators
}
