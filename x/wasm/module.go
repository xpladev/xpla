package wasm

import (
	"encoding/json"
	"fmt"

	appmodule "cosmossdk.io/core/appmodule"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	exported "github.com/cosmos/cosmos-sdk/x/auth/exported"

	upstream "github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/simulation"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xplawasmkeeper "github.com/xpladev/xpla/x/wasm/keeper"
)

type AppModule struct {
	upstream.AppModule

	keeper         *xplawasmkeeper.Keeper
	legacySubspace exported.Subspace
}

var (
	_ appmodule.AppModule        = AppModule{}
	_ module.AppModule           = AppModule{} //nolint:staticcheck // required legacy module-manager compatibility assertion
	_ module.AppModuleSimulation = AppModule{}
)

func NewAppModule(
	cdc codec.Codec,
	keeper *xplawasmkeeper.Keeper,
	validatorSetSource wasmkeeper.ValidatorSetSource,
	accountKeeper wasmtypes.AccountKeeper,
	bankKeeper simulation.BankKeeper,
	router *baseapp.MsgServiceRouter,
	legacySubspace exported.Subspace,
) AppModule {
	return AppModule{
		AppModule: upstream.NewAppModule(
			cdc,
			&keeper.Keeper,
			validatorSetSource,
			accountKeeper,
			bankKeeper,
			router,
			legacySubspace,
		),
		keeper:         keeper,
		legacySubspace: legacySubspace,
	}
}

func (am AppModule) RegisterServices(cfg module.Configurator) {
	wasmtypes.RegisterMsgServer(cfg.MsgServer(), xplawasmkeeper.NewMsgServerImpl(am.keeper))
	wasmtypes.RegisterQueryServer(cfg.QueryServer(), wasmkeeper.Querier(&am.keeper.Keeper))

	migrator := wasmkeeper.NewMigrator(am.keeper.Keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 1, migrator.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", wasmtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 2, migrator.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", wasmtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(wasmtypes.ModuleName, 3, migrator.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", wasmtypes.ModuleName, err))
	}
}

func (am AppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	validatorUpdates := am.AppModule.InitGenesis(ctx, cdc, data)
	if err := am.keeper.RebuildWasmAliases(ctx); err != nil {
		panic(fmt.Sprintf("rebuild wasm aliases after genesis import: %v", err))
	}
	return validatorUpdates
}
