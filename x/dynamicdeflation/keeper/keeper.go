package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

type Keeper struct {
	authority     string
	accountKeeper types.AccountKeeper
	bankKeeper    types.BankKeeper
	distKeeper    types.DistributionKeeper
	poolAddress   sdk.AccAddress

	ParamsStore        collections.Item[types.Params]
	CurrentPeriodStore collections.Item[types.CurrentPeriod]
	Schema             collections.Schema
}

func NewKeeper(
	c codec.Codec,
	storeService store.KVStoreService,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	distKeeper types.DistributionKeeper,
	authority string,
) Keeper {
	poolAddress := accountKeeper.GetModuleAddress(types.PoolName)
	if poolAddress == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.PoolName))
	}

	sb := collections.NewSchemaBuilder(storeService)
	k := Keeper{
		authority:          authority,
		accountKeeper:      accountKeeper,
		bankKeeper:         bankKeeper,
		distKeeper:         distKeeper,
		poolAddress:        poolAddress,
		ParamsStore:        collections.NewItem(sb, types.ParamsPrefix, "params", codec.CollValue[types.Params](c)),
		CurrentPeriodStore: collections.NewItem(sb, types.CurrentPeriodPrefix, "current_period", codec.CollValue[types.CurrentPeriod](c)),
	}
	var err error
	k.Schema, err = sb.Build()
	if err != nil {
		panic(err)
	}
	return k
}

func (k Keeper) Authority() string { return k.authority }

func (k Keeper) Logger(ctx context.Context) log.Logger {
	return sdk.UnwrapSDKContext(ctx).Logger().With("module", "x/"+types.ModuleName)
}
