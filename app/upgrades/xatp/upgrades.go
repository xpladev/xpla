package xatp

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	xatpKeeper "github.com/xpladev/xpla/x/xatp/keeper"
	xatptypes "github.com/xpladev/xpla/x/xatp/types"

	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
)

// CreateUpgradeHandler creates an SDK upgrade handler for xatp
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	xatpKeeper xatpKeeper.Keeper,
	fk feemarketkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {

		vm[xatptypes.ModuleName] = 1

		var msg UpgradeXatpMsg

		err := json.Unmarshal([]byte(plan.Info), &msg)
		if err != nil {
			return nil, sdkerrors.Wrapf(sdkerrors.ErrJSONUnmarshal, UpgradeName+": unmarshal error")
		}

		xatpKeeper.SetParams(ctx, msg.XATP)

		feemarketParams := fk.GetParams(ctx)
		feemarketParams.MinGasPrice = msg.MinGasPrice
		fk.SetParams(ctx, feemarketParams)

		return mm.RunMigrations(ctx, configurator, vm)
	}
}
