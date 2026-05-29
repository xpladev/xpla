package v2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/reward/exported"
	"github.com/xpladev/xpla/x/reward/types"
)

func MigrateStore(ctx sdk.Context, store storetypes.KVStore, legacySubspace exported.Subspace, cdc codec.BinaryCodec) error {
	var currParams types.Params
	legacySubspace.GetParamSet(ctx, &currParams)

	if err := currParams.ValidateBasic(); err != nil {
		return err
	}

	bz, err := cdc.Marshal(&currParams)
	if err != nil {
		return err
	}

	store.Set(types.ParamsKey, bz)

	return nil
}
