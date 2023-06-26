package volunteer

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/keeper"
)

func BeginBlock(ctx sdk.Context, k keeper.Keeper) {
	err := k.VolunteerValidatorCommissionProcess(ctx)
	if err != nil {
		panic(err)
	}
}
