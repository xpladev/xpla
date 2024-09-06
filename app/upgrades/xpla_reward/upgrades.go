package xpla_reward

import (
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/gogoproto/jsonpb"

	"github.com/xpladev/xpla/app/keepers"
	"github.com/xpladev/xpla/x/reward"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		fromVM[rewardtypes.ModuleName] = reward.AppModule{}.ConsensusVersion()

		var params rewardtypes.Params
		err := jsonpb.UnmarshalString(plan.Info, &params)
		if err != nil {
			panic(err)
		}

		keepers.RewardKeeper.SetParams(ctx, params)

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
