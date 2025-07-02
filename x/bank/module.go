package bank

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/bank/exported"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	authkeeper "github.com/xpladev/xpla/x/auth/keeper"
	"github.com/xpladev/xpla/x/bank/keeper"
	banktypesxpla "github.com/xpladev/xpla/x/bank/types"
)

type AppModule struct {
	bank.AppModule

	keeper keeper.Keeper

	// legacySubspace is used solely for migration of x/params managed parameters
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper keeper.Keeper, accountKeeper authkeeper.AccountKeeper, ss exported.Subspace) AppModule {
	return AppModule{
		AppModule:      bank.NewAppModule(cdc, keeper.BaseKeeper, accountKeeper.AccountKeeper, ss),
		keeper:         keeper,
		legacySubspace: ss,
	}
}

// RegisterLegacyAminoCodec registers the bank module's types on the LegacyAmino codec.
func (AppModule) RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	// Register cosmos-sdk bank types
	banktypes.RegisterLegacyAminoCodec(cdc)

	banktypesxpla.RegisterLegacyAminoCodec(cdc)
}

// RegisterInterfaces registers interfaces and implementations of the bank module.
func (AppModule) RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	// Register cosmos-sdk bank interfaces
	banktypes.RegisterInterfaces(registry)

	banktypesxpla.RegisterInterfaces(registry)
}

// RegisterServices registers module services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	// Register cosmos-sdk bank services (Send, MultiSend, etc.)
	banktypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	banktypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)

	// Register XPLA bank services (Burn only)
	banktypesxpla.RegisterMsgServer(cfg.MsgServer(), keeper.NewXplaMsgServerImpl(am.keeper))

	m := bankkeeper.NewMigrator(am.keeper.BaseKeeper, am.legacySubspace)
	if err := cfg.RegisterMigration(banktypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 1 to 2: %v", err))
	}

	if err := cfg.RegisterMigration(banktypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 2 to 3: %v", err))
	}

	if err := cfg.RegisterMigration(banktypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/bank from version 3 to 4: %v", err))
	}
}
