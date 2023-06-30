package volunteer

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	icacontrollertypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	icahosttypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/types"
	icatypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/types"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	routertypes "github.com/strangelove-ventures/packet-forward-middleware/v4/router/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[icatypes.ModuleName] = mm.Modules[icatypes.ModuleName].ConsensusVersion()
		fromVM[routertypes.ModuleName] = mm.Modules[routertypes.ModuleName].ConsensusVersion()
		fromVM[ibcfeetypes.ModuleName] = mm.Modules[ibcfeetypes.ModuleName].ConsensusVersion()

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
		keepers.RouterKeeper.SetParams(ctx, routertypes.DefaultParams())

		return versionMap, err
	}
}
