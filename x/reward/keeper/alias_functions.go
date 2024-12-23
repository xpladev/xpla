package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/xpla/x/reward/types"
)

// GetRewardAccount returns the reward ModuleAccount
func (k Keeper) GetRewardAccount(ctx context.Context) sdk.ModuleAccountI {
	return k.authKeeper.GetModuleAccount(ctx, types.ModuleName)
}

func (k Keeper) GetBlocksPerYear(ctx context.Context) (uint64, error) {
	params, err := k.mintKeeper.Params.Get(ctx)
	return params.BlocksPerYear, err
}

func (k Keeper) PoolBalances(ctx context.Context) sdk.Coins {
	rewardAcc := k.GetRewardAccount(ctx)

	pool := k.bankKeeper.GetAllBalances(ctx, rewardAcc.GetAddress())
	if pool == nil {
		pool = sdk.Coins{}
	}

	return pool
}
