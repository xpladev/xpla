package v1_11

import (
	"context"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"

	"github.com/xpladev/xpla/app/keepers"
)

// legacyPFMParamsKey was used by PFM consensus version 2 to store module
// parameters. PFM v10 removed the parameters but left the value in the module
// store, while the v11 migration interprets every store value as an in-flight
// packet.
var legacyPFMParamsKey = []byte{0x00}

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	appKeepers *keepers.AppKeepers,
	_ codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		removeLegacyPFMParams(ctx, appKeepers)

		// This runs the ibc-go v11.2 in-place migrations on the legacy ibc-apps
		// stores: PFM 3->4 and rate-limiting 1->2. The PFM migration rejects
		// legacy nonrefundable in-flight packets instead of changing their refund
		// semantics during the upgrade.
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func removeLegacyPFMParams(ctx sdk.Context, appKeepers *keepers.AppKeepers) {
	store := ctx.KVStore(appKeepers.GetKVStoreKey()[pfmtypes.StoreKey])
	if !store.Has(legacyPFMParamsKey) {
		return
	}

	store.Delete(legacyPFMParamsKey)
	ctx.Logger().Info("removed legacy PFM params from module store", "module", pfmtypes.ModuleName)
}
