package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestStatusSurplusAndDeficit(t *testing.T) {
	tests := []struct {
		name               string
		balance, allocated int64
		wantSurplus        int64
		wantDeficit        int64
	}{
		{"surplus", 12, 10, 2, 0},
		{"deficit", 8, 10, 0, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newKeeperFixture(t)
			p := types.DefaultParams()
			p.SettlementIntervalBlocks = 2
			period := types.CurrentPeriod{
				StartHeight: 3, EndHeight: 4, ActiveConfig: p,
				GrossAmount: sdkmath.NewInt(10), AllocatedAmount: sdkmath.NewInt(tt.allocated),
			}
			require.NoError(t, f.keeper.CurrentPeriodStore.Set(f.ctx, period))
			f.bank.set(types.PoolName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, tt.balance)))
			res, err := f.keeper.Status(f.ctx, &types.QueryStatusRequest{})
			require.NoError(t, err)
			require.Equal(t, tt.balance, res.ModuleBalance.Amount.Int64())
			require.Equal(t, tt.allocated, res.AllocatedAmount.Amount.Int64())
			require.Equal(t, tt.wantSurplus, res.SurplusAmount.Amount.Int64())
			require.Equal(t, tt.wantDeficit, res.DeficitAmount.Amount.Int64())
		})
	}
}
