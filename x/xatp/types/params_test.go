package types

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	require.NoError(t, DefaultParams().ValidateBasic())
}

func TestParms_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		fields  Params
		wantErr bool
	}{
		{"success", Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(8333, 4), sdk.NewDecWithPrec(1316, 4), sdk.NewDecWithPrec(0, 2), sdk.NewDecWithPrec(333, 4), ""}, false},
		{"empty reserve account with reserver account rate ", Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(8333, 4), sdk.NewDecWithPrec(1316, 4), sdk.NewDecWithPrec(16, 4), sdk.NewDecWithPrec(333, 4), ""}, true},
		{"total rate is more than one", Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(8333, 4), sdk.NewDecWithPrec(1316, 4), sdk.NewDecWithPrec(16, 4), sdk.NewDecWithPrec(336, 4), "aaaa"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.fields.ValidateBasic(); (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
