package volunteercube

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		params := keepers.FeeMarketKeeper.GetParams(ctx)
		params.NoBaseFee = false
		params.BaseFee = sdk.NewInt(850000000000)
		params.BaseFeeChangeDenominator = 1
		params.ElasticityMultiplier = 1
		keepers.FeeMarketKeeper.SetParams(ctx, params)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
