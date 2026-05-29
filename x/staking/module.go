package staking

import (
	"context"
	"fmt"

	"cosmossdk.io/core/appmodule"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/exported"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/xpladev/xpla/x/staking/keeper"
)

var (
	_ module.AppModule          = staking.AppModule{}
	_ module.HasABCIEndBlock    = AppModule{}
	_ appmodule.HasBeginBlocker = AppModule{}
)

type AppModule struct {
	staking.AppModule

	keeper         *keeper.Keeper
	legacySubspace exported.Subspace
}

// NewAppModule creates a new AppModule object
func NewAppModule(cdc codec.Codec, keeper *keeper.Keeper, ak stakingtypes.AccountKeeper, bk stakingtypes.BankKeeper, ls exported.Subspace) AppModule {
	return AppModule{
		AppModule:      staking.NewAppModule(cdc, keeper.Keeper, ak, bk, ls),
		keeper:         keeper,
		legacySubspace: ls,
	}
}

// RegisterServices registers the XPLA staking msg server and SDK query/migration services.
func (am AppModule) RegisterServices(cfg module.Configurator) {
	stakingtypes.RegisterMsgServer(cfg.MsgServer(), keeper.NewMsgServerImpl(am.keeper))
	stakingtypes.RegisterQueryServer(cfg.QueryServer(), stakingkeeper.Querier{Keeper: am.keeper.Keeper})

	m := stakingkeeper.NewMigrator(am.keeper.Keeper, am.legacySubspace)
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 1, m.Migrate1to2); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 1 to 2: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 2, m.Migrate2to3); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 2 to 3: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 3, m.Migrate3to4); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 3 to 4: %v", stakingtypes.ModuleName, err))
	}
	if err := cfg.RegisterMigration(stakingtypes.ModuleName, 4, m.Migrate4to5); err != nil {
		panic(fmt.Sprintf("failed to migrate x/%s from version 4 to 5: %v", stakingtypes.ModuleName, err))
	}
}

// BeginBlock returns the begin blocker for the staking module.
func (am AppModule) BeginBlock(ctx context.Context) error {
	return BeginBlocker(ctx, am.keeper)
}

// EndBlock returns the end blocker for the staking module. It returns no validator
// updates.
func (am AppModule) EndBlock(ctx context.Context) ([]abci.ValidatorUpdate, error) {
	return EndBlocker(ctx, am.keeper)
}
