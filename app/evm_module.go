package app

import (
	"context"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	evm "github.com/cosmos/evm/x/vm"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	xplaprecompile "github.com/xpladev/xpla/precompile"
)

type xplaEVMAppModule struct {
	evm.AppModule

	keeper *evmkeeper.Keeper
}

func newEVMAppModule(app *XplaApp) xplaEVMAppModule {
	return xplaEVMAppModule{
		AppModule: evm.NewAppModule(app.EvmKeeper, app.AccountKeeper, app.BankKeeper, app.AccountKeeper.AddressCodec()),
		keeper:    app.EvmKeeper,
	}
}

func (am xplaEVMAppModule) RegisterServices(cfg module.Configurator) {
	evmtypes.RegisterMsgServer(cfg.MsgServer(), xplaEVMMsgServer{MsgServer: am.keeper})
	evmtypes.RegisterQueryServer(cfg.QueryServer(), am.keeper)
}

func (am xplaEVMAppModule) ValidateGenesis(cdc codec.JSONCodec, txConfig client.TxEncodingConfig, bz json.RawMessage) error {
	if err := am.AppModule.ValidateGenesis(cdc, txConfig, bz); err != nil {
		return err
	}

	var genesis evmtypes.GenesisState
	if err := cdc.UnmarshalJSON(bz, &genesis); err != nil {
		return fmt.Errorf("failed to unmarshal %s genesis state: %w", evmtypes.ModuleName, err)
	}

	return xplaprecompile.ValidateActiveStaticPrecompiles(genesis.Params.ActiveStaticPrecompiles)
}

func (am xplaEVMAppModule) InitGenesis(ctx sdk.Context, cdc codec.JSONCodec, data json.RawMessage) []abci.ValidatorUpdate {
	var genesis evmtypes.GenesisState
	cdc.MustUnmarshalJSON(data, &genesis)
	if err := xplaprecompile.ValidateActiveStaticPrecompiles(genesis.Params.ActiveStaticPrecompiles); err != nil {
		panic(fmt.Errorf("invalid evm genesis active static precompiles: %w", err))
	}

	return am.AppModule.InitGenesis(ctx, cdc, data)
}

type xplaEVMMsgServer struct {
	evmtypes.MsgServer
}

func (s xplaEVMMsgServer) UpdateParams(ctx context.Context, req *evmtypes.MsgUpdateParams) (*evmtypes.MsgUpdateParamsResponse, error) {
	if err := xplaprecompile.ValidateActiveStaticPrecompiles(req.Params.ActiveStaticPrecompiles); err != nil {
		return nil, err
	}

	return s.MsgServer.UpdateParams(ctx, req)
}
