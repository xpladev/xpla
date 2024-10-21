package types_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/reward/types"
)

func TestParms_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		fields  types.Params
		wantErr bool
	}{
		{"success", types.Params{sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.LegacyNewDecWithPrec(80, 2), sdkmath.LegacyNewDecWithPrec(0, 2), "", ""}, false},
		{"empty reserve account with reserver account rate ", types.Params{sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.LegacyNewDecWithPrec(79, 2), sdkmath.LegacyNewDecWithPrec(1, 2), "", ""}, true},
		{"nagative fee pool rate", types.Params{sdkmath.LegacyNewDecWithPrec(-20, 2), sdkmath.LegacyNewDecWithPrec(79, 2), sdkmath.LegacyNewDecWithPrec(0, 2), "", ""}, true},
		{"nagative community pool rate", types.Params{sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.LegacyNewDecWithPrec(-79, 2), sdkmath.LegacyNewDecWithPrec(0, 2), "", ""}, true},
		{"nagative reserve pool rate", types.Params{sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.LegacyNewDecWithPrec(79, 2), sdkmath.LegacyNewDecWithPrec(-1, 2), "aaaa", ""}, true},
		{"total rate is more than one", types.Params{sdkmath.LegacyNewDecWithPrec(20, 2), sdkmath.LegacyNewDecWithPrec(79, 2), sdkmath.LegacyNewDecWithPrec(2, 2), "aaaa", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultParams(t *testing.T) {
	require.NoError(t, types.DefaultParams().ValidateBasic())
}
