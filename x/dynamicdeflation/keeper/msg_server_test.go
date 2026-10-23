package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestMsgUpdateParamsAuthorityValidationAndAtomicReplacement(t *testing.T) {
	f := newKeeperFixture(t)
	server := NewMsgServerImpl(f.keeper)
	updated := types.DefaultParams()
	updated.Enabled = false
	updated.AllocationRate = sdkmath.LegacyOneDec()
	updated.SettlementIntervalBlocks = 12

	_, err := server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "unauthorized", Params: updated})
	require.Error(t, err)
	stored, err := f.keeper.GetParams(f.ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), stored)

	invalid := updated
	invalid.SettlementIntervalBlocks = 0
	_, err = server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: invalid})
	require.Error(t, err)
	stored, err = f.keeper.GetParams(f.ctx)
	require.NoError(t, err)
	require.Equal(t, types.DefaultParams(), stored)

	_, err = server.UpdateParams(f.ctx, &types.MsgUpdateParams{Authority: "authority", Params: updated})
	require.NoError(t, err)
	stored, err = f.keeper.GetParams(f.ctx)
	require.NoError(t, err)
	require.Equal(t, updated, stored)
}
