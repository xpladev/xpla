package wasm

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestChargeFailedExecutionGas(t *testing.T) {
	contract := &vm.Contract{Gas: 100}

	chargeFailedExecutionGas(contract, 30)

	require.Equal(t, uint64(70), contract.Gas)
}

func TestChargeFailedExecutionGasCapsToAvailableGas(t *testing.T) {
	contract := &vm.Contract{Gas: 20}

	chargeFailedExecutionGas(contract, 30)

	require.Zero(t, contract.Gas)
}
