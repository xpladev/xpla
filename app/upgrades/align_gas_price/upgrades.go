package align_gas_price

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	fk feemarketkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		var msg UpgradeAlignGasPriceMsg
		err := json.Unmarshal([]byte(plan.Info), &msg)
		if err != nil {
			panic(err)
		}

		params := fk.GetParams(ctx)
		params.MinGasPrice = msg.MinGasPrice
		fk.SetParams(ctx, params)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
