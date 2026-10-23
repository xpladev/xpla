package common

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/stretchr/testify/require"
)

func TestChargeFailedExecutionGas(t *testing.T) {
	for _, tc := range []struct {
		name                string
		remaining, consumed uint64
		wantGas             uint64
	}{
		{name: "charges consumed gas", remaining: 100, consumed: 30, wantGas: 70},
		{name: "no consumption", remaining: 100, wantGas: 100},
		{name: "caps to remaining gas", remaining: 5, consumed: 30},
		{name: "exact remaining gas", remaining: 30, consumed: 30},
		{name: "no remaining gas", consumed: 30},
	} {
		t.Run(tc.name, func(t *testing.T) {
			contract := &vm.Contract{Gas: tc.remaining}
			ChargeFailedExecutionGas(contract, tc.consumed)
			require.Equal(t, tc.wantGas, contract.Gas)
		})
	}
}
