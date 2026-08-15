package types

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCalculateSettlementEndHeight(t *testing.T) {
	tests := []struct {
		name       string
		start      int64
		interval   uint64
		want       int64
		wantErrMsg string
	}{
		{name: "aligned block", start: 123000, interval: 1000, want: 123000},
		{name: "large aligned block", start: 5123000, interval: 1000, want: 5123000},
		{name: "next boundary", start: 123001, interval: 1000, want: 124000},
		{name: "fresh genesis first period", start: 2, interval: 1000, want: 1000},
		{name: "interval one", start: 37, interval: 1, want: 37},
		{name: "zero interval", start: 2, interval: 0, wantErrMsg: "interval must be between"},
		{name: "invalid start", start: 0, interval: 1000, wantErrMsg: "start height must be positive"},
		{name: "overflow", start: math.MaxInt64, interval: 1000, wantErrMsg: "overflows int64"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CalculateSettlementEndHeight(tt.start, tt.interval)
			if tt.wantErrMsg != "" {
				require.ErrorContains(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
			require.Zero(t, got%int64(tt.interval))
		})
	}
}
