package v1_10

import (
	"context"

	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/app/keepers"
	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		evmParams := keepers.EvmKeeper.GetParams(ctx)
		evmParams.EvmDenom = xplatypes.DefaultDenom
		evmParams.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
			ExtendedDenom: xplatypes.DefaultDenom,
		}
		evmParams.ActiveStaticPrecompiles = xplaprecompile.DefaultActiveStaticPrecompiles()
		if err := keepers.EvmKeeper.SetParams(ctx, evmParams); err != nil {
			return vm, err
		}
		if err := keepers.EvmKeeper.AddPreinstalls(ctx, evmtypes.DefaultPreinstalls); err != nil {
			return vm, err
		}

		return vm, nil
	}
}
