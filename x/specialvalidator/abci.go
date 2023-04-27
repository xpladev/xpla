package specialvalidator

import (
	"time"

	abci "github.com/tendermint/tendermint/abci/types"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/specialvalidator/keeper"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

// Called every block, update validator set
func EndBlocker(ctx sdk.Context, k keeper.Keeper) []abci.ValidatorUpdate {
	defer telemetry.ModuleMeasureSince(types.ModuleName, time.Now(), telemetry.MetricKeyEndBlocker)

	updates, err := k.SpecialValidatorUpdates(ctx)
	if err != nil {
		panic(err)
	}

	return updates
}
