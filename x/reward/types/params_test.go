package types_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/reward/types"
)

func TestParms_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		fields  types.Params
		wantErr bool
	}{
		{"success", types.Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(80, 2), sdk.NewDecWithPrec(0, 2), "", []string{}}, false},
		{"empty reserve account with reserver account rate ", types.Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(79, 2), sdk.NewDecWithPrec(1, 2), "", []string{}}, true},
		{"nagative fee pool rate", types.Params{sdk.NewDecWithPrec(-20, 2), sdk.NewDecWithPrec(79, 2), sdk.NewDecWithPrec(0, 2), "", []string{}}, true},
		{"nagative community pool rate", types.Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(-79, 2), sdk.NewDecWithPrec(0, 2), "", []string{}}, true},
		{"nagative reserve pool rate", types.Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(79, 2), sdk.NewDecWithPrec(-1, 2), "aaaa", []string{}}, true},
		{"total rate is more than one", types.Params{sdk.NewDecWithPrec(20, 2), sdk.NewDecWithPrec(79, 2), sdk.NewDecWithPrec(2, 2), "aaaa", []string{}}, true},
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
