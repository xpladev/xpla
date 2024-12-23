package keeper

import (
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/reward/exported"
	v2 "github.com/xpladev/xpla/x/reward/migrations/v2"
)

// Migrator is a struct for handling in-place store migrations.
type Migrator struct {
	keeper         Keeper
	legacySubspace exported.Subspace
}

// NewMigrator returns a new Migrator.
func NewMigrator(keeper Keeper, legacySubspace exported.Subspace) Migrator {
	return Migrator{keeper: keeper, legacySubspace: legacySubspace}
}

// Migrate1to2 migrates the x/reward module state from the consensus
// version 1 to version 2. Specifically, it takes the parameters that are currently stored
// and managed by the x/params module and stores them directly into the x/reward
// module state.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	store := runtime.KVStoreAdapter(m.keeper.storeService.OpenKVStore(ctx))
	return v2.MigrateStore(ctx, store, m.legacySubspace, m.keeper.cdc)
}
