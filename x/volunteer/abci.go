package volunteer

import (
	"context"

	"github.com/xpladev/xpla/x/volunteer/keeper"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	err := k.VolunteerValidatorCommissionProcess(ctx)
	if err != nil {
		return err
	}
	return nil
}
