package auth

import (
	"embed"
	"errors"

	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"

	"github.com/xpladev/xpla/precompile/util"
	xplatypes "github.com/xpladev/xpla/types"
)

var _ vm.PrecompiledContract = PrecompiledAuth{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IAuth.abi
	abiFS embed.FS
)

type PrecompiledAuth struct {
	cmn.Precompile
	ak AccountKeeper
}

func init() {
	var err error
	ABI, err = util.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledAuth(ak AccountKeeper) PrecompiledAuth {
	p := PrecompiledAuth{
		Precompile: cmn.Precompile{
			ABI:                  ABI,
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		ak: ak,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledAuth) RequiredGas(input []byte) uint64 {
	// Implement the method as needed
	return 0
}

func (p PrecompiledAuth) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, argsBz := util.SplitInput(contract.Input)

	abiMethod, err := ABI.MethodById(method)
	if err != nil {
		return nil, err
	}

	args, err := abiMethod.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}

	ctx := evm.StateDB.(*statedb.StateDB).GetContext()

	switch MethodAuth(abiMethod.Name) {
	case Account:
		return p.account(ctx, abiMethod, args)
	case ModuleAccountByName:
		return p.moduleAccountByName(ctx, abiMethod, args)
	case Bech32Prefix:
		return p.bech32Prefix(ctx, abiMethod, args)
	case AddressBytesToString:
		return p.addressBytesToString(ctx, abiMethod, args)
	case AddressStringToBytes:
		return p.addressStringToBytes(ctx, abiMethod, args)
	default:
		return nil, errors.New("method not found")
	}
}

func (p PrecompiledAuth) account(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	address, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	var strAddress string
	if p.ak.HasAccount(ctx, address) {
		// address: contract or address
		account := p.ak.GetAccount(ctx, address)
		strAddress = account.GetAddress().String()
	} else {
		// cannot query
		strAddress = ""
	}

	return method.Outputs.Pack(strAddress)
}

func (p PrecompiledAuth) moduleAccountByName(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	moduleName, err := util.GetString(args[0])
	if err != nil {
		return nil, err
	}

	account := p.ak.GetModuleAccount(ctx, moduleName)
	if account == nil {
		return method.Outputs.Pack("")
	} else {
		return method.Outputs.Pack(account.GetAddress().String())
	}
}

func (p PrecompiledAuth) bech32Prefix(_ sdk.Context, method *abi.Method, _ []interface{}) ([]byte, error) {
	return method.Outputs.Pack(xplatypes.Bech32MainPrefix)
}

func (p PrecompiledAuth) addressBytesToString(_ sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	address, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(address.String())
}

func (p PrecompiledAuth) addressStringToBytes(_ sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	stringAddress, err := util.GetString(args[0])
	if err != nil {
		return nil, err
	}

	byteAddress, err := sdk.AccAddressFromBech32(stringAddress)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(ethcommon.BytesToAddress(byteAddress.Bytes()))
}
