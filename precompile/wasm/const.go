package wasm

const (
	hexAddress = "0x1000000000000000000000000000000000000004"
	delegatecallHexAddress = "0x1000000000000000000000000000000000000044"
	abiFile    = "IWasm.abi"
)

type MethodWasm string

const (
	InstantiateContract  MethodWasm = "instantiateContract"
	InstantiateContract2 MethodWasm = "instantiateContract2"
	ExecuteContract      MethodWasm = "executeContract"
	MigrateContract      MethodWasm = "migrateContract"

	SmartContractState MethodWasm = "smartContractState"
)
