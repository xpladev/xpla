package v1_11

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/xpladev/xpla/app/keepers"
)

// CreateUpgradeHandler creates the v1_11 upgrade handler. The EVM hotfix does
// not change module consensus versions or store layouts, so the standard module
// migrations are sufficient for this binary upgrade.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	_ *keepers.AppKeepers,
	_ codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
