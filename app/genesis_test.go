package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestDefaultGenesisUsesXplaDenom(t *testing.T) {
	xpla := newTestApp(t)
	genesis := xpla.DefaultGenesis()

	var bankGenesis banktypes.GenesisState
	xpla.appCodec.MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGenesis)
	require.Len(t, bankGenesis.DenomMetadata, 1)
	require.Equal(t, xplatypes.DefaultDenom, bankGenesis.DenomMetadata[0].Base)
	require.Equal(t, "xpla", bankGenesis.DenomMetadata[0].Display)
	require.Equal(t, "XPLA", bankGenesis.DenomMetadata[0].Symbol)

	var stakingGenesis stakingtypes.GenesisState
	xpla.appCodec.MustUnmarshalJSON(genesis[stakingtypes.ModuleName], &stakingGenesis)
	require.Equal(t, xplatypes.DefaultDenom, stakingGenesis.Params.BondDenom)

	var mintGenesis minttypes.GenesisState
	xpla.appCodec.MustUnmarshalJSON(genesis[minttypes.ModuleName], &mintGenesis)
	require.Equal(t, xplatypes.DefaultDenom, mintGenesis.Params.MintDenom)

	var govGenesis govv1types.GenesisState
	xpla.appCodec.MustUnmarshalJSON(genesis[govtypes.ModuleName], &govGenesis)
	require.NotNil(t, govGenesis.Params)
	for _, deposit := range govGenesis.Params.MinDeposit {
		require.Equal(t, xplatypes.DefaultDenom, deposit.Denom)
	}
	for _, deposit := range govGenesis.Params.ExpeditedMinDeposit {
		require.Equal(t, xplatypes.DefaultDenom, deposit.Denom)
	}

	var evmGenesis evmtypes.GenesisState
	xpla.appCodec.MustUnmarshalJSON(genesis[evmtypes.ModuleName], &evmGenesis)
	require.Equal(t, xplatypes.DefaultDenom, evmGenesis.Params.EvmDenom)
	require.NotNil(t, evmGenesis.Params.ExtendedDenomOptions)
	require.Equal(t, xplatypes.DefaultDenom, evmGenesis.Params.ExtendedDenomOptions.ExtendedDenom)
	require.Equal(t, xplaprecompile.DefaultActiveStaticPrecompiles(), evmGenesis.Params.ActiveStaticPrecompiles)
	require.Contains(t, evmGenesis.Params.ActiveStaticPrecompiles, evmtypes.ICS20PrecompileAddress)
	require.Contains(t, evmGenesis.Params.ActiveStaticPrecompiles, evmtypes.ICS02PrecompileAddress)
	require.Equal(t, evmtypes.DefaultPreinstalls, evmGenesis.Preinstalls)
}
