package v2

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	xatpKeeper "github.com/xpladev/xpla/x/xatp/keeper"
	xatptypes "github.com/xpladev/xpla/x/xatp/types"
)

// CreateUpgradeHandler creates an SDK upgrade handler for v2
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	xatpKeeper xatpKeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		vm[xatptypes.ModuleName] = 1

		var params xatptypes.Params

		err := json.Unmarshal([]byte(plan.Info), &params)
		if err != nil {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrJSONUnmarshal, UpgradeName+": unmarshal error")
		}

		xatpKeeper.SetParams(ctx, params)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
