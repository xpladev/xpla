package common

import (
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ChargeFailedExecutionGas charges native execution gas on an ordinary error.
// RunNativeAction currently charges only successful actions; remove or adjust
// this helper if upstream starts charging ordinary errors.
func ChargeFailedExecutionGas(contract *vm.Contract, gas uint64) {
	if gas > contract.Gas {
		gas = contract.Gas
	}
	contract.UseGas(gas, nil, tracing.GasChangeCallFailedExecution)
}
