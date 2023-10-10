package evm

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/evmos/evmos/v14/x/evm"
	"github.com/evmos/evmos/v14/x/evm/keeper"
	"github.com/evmos/evmos/v14/x/evm/types"
)

var (
	_ module.AppModule = AppModule{}
)

type AppModule struct {
	evm.AppModule

	keeper *keeper.Keeper
	ak     types.AccountKeeper
}

// NewAppModule creates a new AppModule object
func NewAppModule(
	keeper *keeper.Keeper,
	ak types.AccountKeeper,
	ss types.Subspace) AppModule {
	return AppModule{
		AppModule: evm.NewAppModule(keeper, ak, ss),
		keeper:    keeper,
		ak:        ak,
	}
}

// ExportGenesis returns the exported genesis state as raw bytes for the evm
// module.
func (am AppModule) ExportGenesis(ctx sdk.Context, cdc codec.JSONCodec) json.RawMessage {
	gs := ExportGenesis(ctx, am.keeper, am.ak)
	return cdc.MustMarshalJSON(gs)
}
