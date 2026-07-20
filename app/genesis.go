package app

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

// The genesis state of the blockchain is represented here as a map of raw json
// messages key'd by a identifier string.
// The identifier is used to determine which module genesis information belongs
// to so it may be appropriately routed during init chain.
// Within this application default genesis information is retrieved from
// the ModuleBasicManager which populates json from each BasicModule
// object provided to it during init.
type GenesisState map[string]json.RawMessage

// DefaultGenesis returns a default genesis from the registered AppModuleBasic's.
func (app *XplaApp) DefaultGenesis() map[string]json.RawMessage {
	genesis := app.ModuleBasics.DefaultGenesis(app.appCodec)
	genesis[banktypes.ModuleName] = app.appCodec.MustMarshalJSON(NewBankGenesisState())
	genesis[stakingtypes.ModuleName] = app.appCodec.MustMarshalJSON(NewStakingGenesisState(app, genesis))
	genesis[minttypes.ModuleName] = app.appCodec.MustMarshalJSON(NewMintGenesisState(app, genesis))
	genesis[govtypes.ModuleName] = app.appCodec.MustMarshalJSON(NewGovGenesisState(app, genesis))
	genesis[evmtypes.ModuleName] = app.appCodec.MustMarshalJSON(NewEVMGenesisState(app, genesis))

	return genesis
}

func NewBankGenesisState() *banktypes.GenesisState {
	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.DenomMetadata = XplaDenomMetadata()

	return bankGenesis
}

func NewStakingGenesisState(app *XplaApp, genesis GenesisState) *stakingtypes.GenesisState {
	var stakingGenesis stakingtypes.GenesisState
	app.appCodec.MustUnmarshalJSON(genesis[stakingtypes.ModuleName], &stakingGenesis)
	stakingGenesis.Params.BondDenom = xplatypes.DefaultDenom

	return &stakingGenesis
}

func NewMintGenesisState(app *XplaApp, genesis GenesisState) *minttypes.GenesisState {
	var mintGenesis minttypes.GenesisState
	app.appCodec.MustUnmarshalJSON(genesis[minttypes.ModuleName], &mintGenesis)
	mintGenesis.Params.MintDenom = xplatypes.DefaultDenom

	return &mintGenesis
}

func NewGovGenesisState(app *XplaApp, genesis GenesisState) *govv1types.GenesisState {
	var govGenesis govv1types.GenesisState
	app.appCodec.MustUnmarshalJSON(genesis[govtypes.ModuleName], &govGenesis)
	if govGenesis.Params != nil {
		govGenesis.Params.MinDeposit = coinsWithXplaDenom(govGenesis.Params.MinDeposit)
		govGenesis.Params.ExpeditedMinDeposit = coinsWithXplaDenom(govGenesis.Params.ExpeditedMinDeposit)
	}

	return &govGenesis
}

func NewEVMGenesisState(app *XplaApp, genesis GenesisState) *evmtypes.GenesisState {
	var evmGenesis evmtypes.GenesisState
	app.appCodec.MustUnmarshalJSON(genesis[evmtypes.ModuleName], &evmGenesis)
	evmGenesis.Params.EvmDenom = xplatypes.DefaultDenom
	evmGenesis.Params.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
		ExtendedDenom: xplatypes.DefaultDenom,
	}
	evmGenesis.Params.ActiveStaticPrecompiles = xplaprecompile.DefaultActiveStaticPrecompiles()
	evmGenesis.Preinstalls = evmtypes.DefaultPreinstalls

	return &evmGenesis
}

func coinsWithXplaDenom(coins sdk.Coins) sdk.Coins {
	updated := make(sdk.Coins, 0, len(coins))
	for _, coin := range coins {
		coin.Denom = xplatypes.DefaultDenom
		updated = append(updated, coin)
	}

	return updated
}

func XplaDenomMetadata() []banktypes.Metadata {
	return []banktypes.Metadata{
		{
			Description: "The native staking token for XPLA.",
			Base:        xplatypes.DefaultDenom,
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    xplatypes.DefaultDenom,
					Exponent: 0,
					Aliases:  []string{"attoxpla"},
				},
				{
					Denom:    "xpla",
					Exponent: uint32(xplatypes.DefaultDenomPrecision),
				},
			},
			Display: "xpla",
			Name:    "XPLA Token",
			Symbol:  "XPLA",
		},
	}
}
