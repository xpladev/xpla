package volunteer

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/keeper"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func BeginBlock(ctx sdk.Context, k keeper.Keeper) {
	err := k.VolunteerValidatorCommissionProcess(ctx)
	if err != nil {
		panic(err)
	}
}

// Called every block, update validator set
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	updates, err := k.VolunteerValidatorUpdates(ctx)
	if err != nil {
		panic(err)
	}

	return updates
}
