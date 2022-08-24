package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/xpladev/xpla/x/reward/types"
)

// GetRewardAccount returns the reward ModuleAccount
func (k Keeper) GetRewardAccount(ctx sdk.Context) authtypes.ModuleAccountI {
	return k.authKeeper.GetModuleAccount(ctx, types.ModuleName)
}

func (k Keeper) GetBlocksPerYear(ctx sdk.Context) uint64 {
	params := k.mintKeeper.GetParams(ctx)
	return params.BlocksPerYear
}
