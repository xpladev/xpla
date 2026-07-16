package v1_11

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ *keepers.AppKeepers,
	_ codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// This runs the ibc-go v11.2 in-place migrations on the legacy ibc-apps
		// stores: PFM 3->4 and rate-limiting 1->2. The PFM migration rejects
		// legacy nonrefundable in-flight packets instead of changing their refund
		// semantics during the upgrade.
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
