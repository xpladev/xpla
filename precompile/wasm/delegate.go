package wasm

import "github.com/ethereum/go-ethereum/core/vm"

var _ vm.PrecompiledContract = DelegatePrecompile{}

// DelegatePrecompile routes calls through Wasm delegate execution.
type DelegatePrecompile struct {
	*PrecompiledWasm
}

// NewDelegatePrecompile shares the provided Wasm precompile for delegate execution.
func NewDelegatePrecompile(precompile *PrecompiledWasm) DelegatePrecompile {
	return DelegatePrecompile{PrecompiledWasm: precompile}
}

func (p DelegatePrecompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	return p.PrecompiledWasm.RunDelegate(evm, contract, readOnly)
}
