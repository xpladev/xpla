package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/reward/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	k.SetParams(ctx, data.Params)

	// check if the module account exists
	moduleAcc := k.GetRewardAccount(ctx)
	if moduleAcc == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	balances := k.bankKeeper.GetAllBalances(ctx, moduleAcc.GetAddress())
	if balances.IsZero() {
		k.authKeeper.SetModuleAccount(ctx, moduleAcc)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params := k.GetParams(ctx)
	return types.NewGenesisState(params)
}
