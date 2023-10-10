package evm

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/evmos/evmos/v14/x/evm"
	evmkeeper "github.com/evmos/evmos/v14/x/evm/keeper"
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	"github.com/evmos/evmos/v14/x/feemarket"
	feemarketkeeper "github.com/evmos/evmos/v14/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/evmos/v14/x/feemarket/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	ek evmkeeper.Keeper,
	fk feemarketkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[evmtypes.ModuleName] = evm.AppModule{}.ConsensusVersion()
		fromVM[feemarkettypes.ModuleName] = feemarket.AppModule{}.ConsensusVersion()

		var params EvmUpgradeParams
		err := json.Unmarshal([]byte(plan.Info), &params)
		if err != nil {
			panic(err)
		}

		ek.SetParams(ctx, params.Evm)
		fk.SetParams(ctx, params.FeeMarket)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
