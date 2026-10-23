package dynamicdeflation

import (
	"context"

	"github.com/xpladev/xpla/x/dynamicdeflation/keeper"
)

func BeginBlocker(ctx context.Context, k keeper.Keeper) error {
	return k.BeginBlock(ctx)
}
