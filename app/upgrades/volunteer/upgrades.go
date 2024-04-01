package volunteer

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	router "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward"
	routertypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v7/packetforward/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[icatypes.ModuleName] = ica.AppModule{}.ConsensusVersion()
		fromVM[routertypes.ModuleName] = router.AppModule{}.ConsensusVersion()
		fromVM[ibcfeetypes.ModuleName] = ibcfee.AppModule{}.ConsensusVersion()

		params := keepers.FeeMarketKeeper.GetParams(ctx)
		params.NoBaseFee = false
		params.MinGasPrice = sdk.MustNewDecFromStr("850000000000")
		params.BaseFee = sdk.NewInt(850000000000)
		params.BaseFeeChangeDenominator = 1
		params.ElasticityMultiplier = 1
		keepers.FeeMarketKeeper.SetParams(ctx, params)

		// Run migrations
		versionMap, err := mm.RunMigrations(ctx, configurator, fromVM)

		// update ICA Host to add new messages available
		// enumerate all because it's easier to reason about
		newIcaHostParams := icahosttypes.Params{
			HostEnabled:   true,
			AllowMessages: []string{"*"},
		}
		keepers.ICAHostKeeper.SetParams(ctx, newIcaHostParams)
		keepers.ICAControllerKeeper.SetParams(ctx, icacontrollertypes.Params{ControllerEnabled: true})
		keepers.PFMRouterKeeper.SetParams(ctx, routertypes.DefaultParams())

		return versionMap, err
	}
}
