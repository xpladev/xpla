package types

import (
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func Test_validateAuxFuncs(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"wrong type", args{10.5}, true},
		{"empty sdk.Dec", args{sdk.Dec{}}, true},
		{"negative", args{sdk.NewDec(-1)}, true},
		{"one dec", args{sdk.NewDec(1)}, false},
		{"two dec", args{sdk.NewDec(2)}, true},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantErr, validateFeePoolRate(tt.args.i) != nil)
			require.Equal(t, tt.wantErr, validateCommunityPoolRate(tt.args.i) != nil)
			require.Equal(t, tt.wantErr, validateReserveRate(tt.args.i) != nil)
		})
	}
}

func Test_validateAccountFuncs(t *testing.T) {
	type args struct {
		i interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"wrong type", args{10.5}, true},
		{"empty string", args{""}, false},
		{"invalid bech", args{"a"}, true},
		{"valid", args{"cosmos1qqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqqnrql8a"}, false},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.wantErr, validateAccount(tt.args.i) != nil)
		})
	}
}
