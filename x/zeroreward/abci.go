package zeroreward

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/zeroreward/keeper"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func BeginBlock(ctx sdk.Context, k keeper.Keeper) {
	err := k.ZeroRewardValidatorCommissionProcess(ctx)
	if err != nil {
		panic(err)
	}
}

// Called every block, update validator set
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	updates, err := k.ZeroRewardValidatorUpdates(ctx)
	if err != nil {
		panic(err)
	}

	return updates
}
