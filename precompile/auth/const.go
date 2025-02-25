package auth

const (
	hexAddress = "0x1000000000000000000000000000000000000005"
	abiFile    = "IAuth.abi"
)

type MethodAuth string

const (
	Account              MethodAuth = "account"
	ModuleAccountByName  MethodAuth = "moduleAccountByName"
	Bech32Prefix         MethodAuth = "bech32Prefix"
	AddressBytesToString MethodAuth = "addressBytesToString"
	AddressStringToBytes MethodAuth = "addressStringToBytes"
)
