package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalValidate(t *testing.T) {
	params := Params{
		Payer: "",
	}
	err := params.Validate()
	require.NoError(t, err)
}

func TestFailedValidateWithUnexpectedAddress(t *testing.T) {
	params := Params{
		Payer: "1",
	}
	err := params.Validate()
	require.Error(t, err)
}

func TestNormalValidatePayer(t *testing.T) {
	address := ""
	err := validatePayer(address)
	require.NoError(t, err)
}

func TestFailedValidatePayerWithUnknonwType(t *testing.T) {
	address := 1
	err := validatePayer(address)
	require.Error(t, err)
}

func TestFailedValidateXplaPayerWithUnexpectedAddress(t *testing.T) {
	address := "123"
	err := validatePayer(address)
	require.Error(t, err)
}
