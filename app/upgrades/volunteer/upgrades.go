package volunteer

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ica "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts"
	icacontrollertypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v7/modules/apps/27-interchain-accounts/types"
	ibcfee "github.com/cosmos/ibc-go/v7/modules/apps/29-fee"
	ibcfeetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	"github.com/strangelove-ventures/packet-forward-middleware/v7/router"
	routertypes "github.com/strangelove-ventures/packet-forward-middleware/v7/router/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[icatypes.ModuleName] = ica.AppModule{}.ConsensusVersion()
		fromVM[routertypes.ModuleName] = router.AppModule{}.ConsensusVersion()
		fromVM[ibcfeetypes.ModuleName] = ibcfee.AppModule{}.ConsensusVersion()

		var msg UpgradeVolunteerMsg
		err := json.Unmarshal([]byte(plan.Info), &msg)
		if err != nil {
			panic(err)
		}

		params := keepers.FeeMarketKeeper.GetParams(ctx)
		params.MinGasPrice = msg.MinGasPrice
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
