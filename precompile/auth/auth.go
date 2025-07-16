package auth

import (
	"embed"
	"errors"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
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

func (p PrecompiledAuth) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	// Start the balance change handler before executing the precompile.
	p.GetBalanceHandler().BeforeBalanceChange(ctx)

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

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
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost, nil, tracing.GasChangeCallPrecompiledContract) {
		return cmn.ReturnRevertError(evm, vm.ErrOutOfGas)
	}

	// Process the native balance changes after the method execution.
	if err = p.GetBalanceHandler().AfterBalanceChange(ctx, stateDB); err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	return bz, nil
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
