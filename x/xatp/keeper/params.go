package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func (k Keeper) GetParams(ctx sdk.Context) (params types.Params) {
	k.paramSpace.GetParamSet(ctx, &params)
	return params
}

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	k.paramSpace.SetParamSet(ctx, &params)
}

func (k Keeper) GetXTPSPayer(ctx sdk.Context) (xplaPayer string) {

	k.paramSpace.Get(ctx, types.ParamStoreKeyXATPPayer, &xplaPayer)
	return xplaPayer
}

func (k Keeper) GetXATPs(ctx sdk.Context) (xatps []types.XATP) {

	k.paramSpace.Get(ctx, types.ParamStoreKeyXATPs, &xatps)
	return xatps
}
