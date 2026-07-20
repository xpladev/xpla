package auth

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/exported"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/xpladev/xpla/x/auth/keeper"
)

type AppModule struct {
	auth.AppModule

	accountKeeper keeper.AccountKeeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, accountKeeper keeper.AccountKeeper, randGenAccountsFn authtypes.RandomGenesisAccountsFn, ss exported.Subspace) AppModule {
	return AppModule{
		AppModule:      auth.NewAppModule(cdc, accountKeeper.AccountKeeper, randGenAccountsFn, ss),
		accountKeeper:  accountKeeper,
		legacySubspace: ss,
	}
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	authtypes.RegisterMsgServer(cfg.MsgServer(), authkeeper.NewMsgServerImpl(am.accountKeeper.AccountKeeper))
	authtypes.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.accountKeeper))

	m := authkeeper.NewMigrator(am.accountKeeper.AccountKeeper, cfg.QueryServer(), am.legacySubspace)
	if err := cfg.RegisterMigration(authtypes.ModuleName, 5, m.Migrate5to6); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 5 to 6: %v", authtypes.ModuleName, err))
	}
}
