package types

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestIsNilOrNegativeInt(t *testing.T) {
	tests := []struct {
		name  string
		value sdkmath.Int
		want  bool
	}{
		{name: "nil", value: sdkmath.Int{}, want: true},
		{name: "negative", value: sdkmath.NewInt(-1), want: true},
		{name: "zero", value: sdkmath.ZeroInt(), want: false},
		{name: "positive", value: sdkmath.OneInt(), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsNilOrNegativeInt(tt.value))
		})
	}
}
