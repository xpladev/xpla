package bank

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
)

var _ vm.PrecompiledContract = PrecompiledBank{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IBank.abi
	abiFS embed.FS
)

type PrecompiledBank struct {
	cmn.Precompile
	bk BankKeeper
}

func init() {
	var err error
	ABI, err = util.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledBank(bk BankKeeper) PrecompiledBank {
	p := PrecompiledBank{
		Precompile: cmn.Precompile{
			ABI:                  ABI,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		bk: bk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledBank) Address() common.Address { return Address }

func (p PrecompiledBank) RequiredGas(input []byte) uint64 {
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

func (p PrecompiledBank) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	// Start the balance change handler before executing the precompile.
	p.GetBalanceHandler().BeforeBalanceChange(ctx)

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch MethodBank(method.Name) {
	case Balance:
		bz, err = p.balance(ctx, method, args)
	case Send:
		bz, err = p.send(ctx, evm.Origin, method, args)
	case Supply:
		bz, err = p.supplyOf(ctx, method, args)
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

func (p PrecompiledBank) IsTransaction(method *abi.Method) bool {
	switch MethodBank(method.Name) {
	case Send:
		return true
	default:
		return false
	}
}

func (p PrecompiledBank) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("xpla evm extension", "bank")
}

func (p PrecompiledBank) balance(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {

	address, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	denom, err := util.GetString(args[1])
	if err != nil {
		return nil, err
	}

	coin := p.bk.GetBalance(ctx, address, denom)

	return method.Outputs.Pack(coin.Amount.BigInt())
}

func (p PrecompiledBank) supplyOf(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	denom, err := util.GetString(args[0])
	if err != nil {
		return nil, err
	}

	coin := p.bk.GetSupply(ctx, denom)

	return method.Outputs.Pack(coin.Amount.BigInt())
}

func (p PrecompiledBank) send(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

	fromAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(fromAddress, sender); err != nil {
		return nil, err
	}

	toAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	coins, err := util.GetCoins(args[2])
	if err != nil {
		return nil, err
	}

	err = p.bk.SendCoins(ctx, fromAddress, toAddress, coins)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}
