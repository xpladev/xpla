package keeper

import (
	"math"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestGenesisRoundTrip(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 4
	period := &types.CurrentPeriod{
		StartHeight: 14, EndHeight: 16, ActiveConfig: p,
		GrossAmount: sdkmath.NewInt(7), AllocatedAmount: sdkmath.NewInt(1),
	}
	state := &types.GenesisState{Params: p, CurrentPeriod: period}
	require.NoError(t, state.Validate())
	require.NoError(t, f.keeper.InitGenesis(f.ctx, state))
	exported, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.Equal(t, state, exported)
}

func TestGenesisRejectsImpossibleCurrentPeriod(t *testing.T) {
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 1

	disabledPeriod := types.CurrentPeriod{
		StartHeight:     2,
		EndHeight:       2,
		ActiveConfig:    p,
		GrossAmount:     sdkmath.OneInt(),
		AllocatedAmount: sdkmath.ZeroInt(),
	}
	disabledPeriod.ActiveConfig.Enabled = false
	state := types.GenesisState{Params: p, CurrentPeriod: &disabledPeriod}
	require.ErrorContains(t, state.Validate(), "must be enabled")

	overfundedPeriod := disabledPeriod
	overfundedPeriod.ActiveConfig.Enabled = true
	overfundedPeriod.AllocatedAmount = sdkmath.NewInt(2)
	state = types.GenesisState{Params: p, CurrentPeriod: &overfundedPeriod}
	require.ErrorContains(t, state.Validate(), "allocated amount cannot exceed gross amount")

	p.SettlementIntervalBlocks = 2
	overflowPeriod := types.CurrentPeriod{
		StartHeight:     math.MaxInt64,
		EndHeight:       math.MaxInt64,
		ActiveConfig:    p,
		GrossAmount:     sdkmath.OneInt(),
		AllocatedAmount: sdkmath.ZeroInt(),
	}
	state = types.GenesisState{Params: p, CurrentPeriod: &overflowPeriod}
	require.ErrorContains(t, state.Validate(), "overflows int64")
}

func TestGenesisRejectsHeightOnePeriod(t *testing.T) {
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 1
	period := types.CurrentPeriod{
		StartHeight:     1,
		EndHeight:       1,
		ActiveConfig:    p,
		GrossAmount:     sdkmath.OneInt(),
		AllocatedAmount: sdkmath.ZeroInt(),
	}
	state := types.GenesisState{Params: p, CurrentPeriod: &period}
	require.ErrorContains(t, state.Validate(), "invalid period height range")
}
