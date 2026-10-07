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
	"github.com/xpladev/xpla/x/auth/types"
)

// AppModule extends the SDK auth module only at the account Query boundary.
type AppModule struct {
	auth.AppModule

	accountKeeper  keeper.AccountKeeper
	wasmKeeper     types.WasmKeeper
	legacySubspace exported.Subspace
}

var _ module.HasServices = AppModule{}

func NewAppModule(cdc codec.Codec, accountKeeper keeper.AccountKeeper, randGenAccountsFn authtypes.RandomGenesisAccountsFn, ss exported.Subspace, wasmKeeper types.WasmKeeper) AppModule {
	return AppModule{
		AppModule:      auth.NewAppModule(cdc, accountKeeper.AccountKeeper, randGenAccountsFn, ss),
		accountKeeper:  accountKeeper,
		wasmKeeper:     wasmKeeper,
		legacySubspace: ss,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	authtypes.RegisterMsgServer(cfg.MsgServer(), authkeeper.NewMsgServerImpl(am.accountKeeper.AccountKeeper))
	authtypes.RegisterQueryServer(cfg.QueryServer(), keeper.NewQueryServer(am.accountKeeper, am.wasmKeeper))

	m := authkeeper.NewMigrator(am.accountKeeper.AccountKeeper, cfg.QueryServer(), am.legacySubspace)
	if err := cfg.RegisterMigration(authtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", authtypes.ModuleName, err))
	}

	if err := cfg.RegisterMigration(authtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", authtypes.ModuleName, err))
	}

	if err := cfg.RegisterMigration(authtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", authtypes.ModuleName, err))
	}

	if err := cfg.RegisterMigration(authtypes.ModuleName, 4, m.Migrate4To5); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 4 to 5", authtypes.ModuleName))
	}
}
