package xpla_reward

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/cosmos/gogoproto/jsonpb"
	"github.com/xpladev/xpla/x/reward"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	rk rewardkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[rewardtypes.ModuleName] = reward.AppModule{}.ConsensusVersion()

		var params rewardtypes.Params
		err := jsonpb.UnmarshalString(plan.Info, &params)
		if err != nil {
			panic(err)
		}

		rk.SetParams(ctx, params)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
