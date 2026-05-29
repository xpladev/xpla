package auth

import (
	"bytes"
	"errors"

	_ "embed"

	"cosmossdk.io/log/v2"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/xpladev/xpla/precompile/util"
	xplatypes "github.com/xpladev/xpla/types"
)

var _ vm.PrecompiledContract = PrecompiledAuth{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IAuth.json
	f []byte
)

type PrecompiledAuth struct {
	cmn.Precompile
	abi.ABI
	ak AccountKeeper
}

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledAuth(ak AccountKeeper) PrecompiledAuth {
	p := PrecompiledAuth{
		Precompile: cmn.Precompile{
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		ABI: ABI,
		ak:  ak,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (PrecompiledAuth) Name() string {
	return "auth"
}

func (p PrecompiledAuth) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p PrecompiledAuth) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) (bz []byte, err error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, evm.StateDB, contract, readonly)
	})
}

func (p PrecompiledAuth) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	var bz []byte

	switch MethodAuth(method.Name) {
	case Account:
		bz, err = p.account(ctx, method, args)
	case ModuleAccountByName:
		bz, err = p.moduleAccountByName(ctx, method, args)
	case Bech32Prefix:
		bz, err = p.bech32Prefix(ctx, method, args)
	case AddressBytesToString:
		bz, err = p.addressBytesToString(ctx, method, args)
	case AddressStringToBytes:
		bz, err = p.addressStringToBytes(ctx, method, args)
	default:
		bz, err = nil, errors.New("method not found")
	}

	return bz, err
}

func (p PrecompiledAuth) IsTransaction(method *abi.Method) bool {
	return false
}

func (p PrecompiledAuth) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("xpla evm extension", "auth")
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

	return method.Outputs.Pack(common.BytesToAddress(byteAddress.Bytes()))
}
