package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func (k Keeper) GetXatp(ctx sdk.Context, denom string) (xatp types.XATP, found bool) {
	store := ctx.KVStore(k.storeKey)

	value := store.Get(types.GetXatpKey(denom))
	if value == nil {
		return xatp, false
	}

	xatp = types.MustUnmarshalXatp(k.cdc, value)
	return xatp, true
}

func (k Keeper) SetXatp(ctx sdk.Context, xatp types.XATP) {
	store := ctx.KVStore(k.storeKey)
	bz := types.MustMarshalXatp(k.cdc, &xatp)
	store.Set(types.GetXatpKey(xatp.Denom), bz)
}

func (k Keeper) GetAllXatps(ctx sdk.Context) (xatps []types.XATP) {
	store := ctx.KVStore(k.storeKey)
	iterator := sdk.KVStorePrefixIterator(store, types.XatpsKey)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		xatp := types.MustUnmarshalXatp(k.cdc, iterator.Value())
		xatps = append(xatps, xatp)
	}

	return xatps
}
