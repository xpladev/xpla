package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/reward/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	err := k.SetParams(ctx, data.Params)
	if err != nil {
		panic(fmt.Errorf("error setting params %s", err))
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	params, err := k.GetParams(ctx)
	if err != nil {
		panic(fmt.Errorf("error getting params %s", err))
	}
	return types.NewGenesisState(params)
}
