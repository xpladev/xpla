package bank

import (
	"bytes"
	"errors"
	"fmt"

	_ "embed"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/xpladev/xpla/precompile/util"
)

var _ vm.PrecompiledContract = PrecompiledBank{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IBank.json
	f []byte
)

type TotalSupplyInput struct {
	PageRequest query.PageRequest
}

type PrecompiledBank struct {
	cmn.Precompile
	abi.ABI
	bk BankKeeper
}

func init() {
	var err error
	ABI, err = abi.JSON(bytes.NewReader(f))
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledBank(bk BankKeeper) PrecompiledBank {
	p := PrecompiledBank{
		Precompile: cmn.Precompile{
			KvGasConfig:           storetypes.KVGasConfig(),
			TransientKVGasConfig:  storetypes.TransientGasConfig(),
			BalanceHandlerFactory: cmn.NewBalanceHandlerFactory(bk),
		},
		ABI: ABI,
		bk:  bk,
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

func (p PrecompiledBank) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) (bz []byte, err error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		return p.Execute(ctx, evm.StateDB, contract, readonly)
	})
}

func (p PrecompiledBank) Execute(ctx sdk.Context, stateDB vm.StateDB, contract *vm.Contract, readOnly bool) ([]byte, error) {
	method, args, err := cmn.SetupABI(p.ABI, contract, readOnly, p.IsTransaction)
	if err != nil {
		return nil, err
	}

	var bz []byte

	switch MethodBank(method.Name) {
	case Balance:
		bz, err = p.balance(ctx, method, args)
	case Send:
		bz, err = p.send(ctx, stateDB, contract.Caller(), method, args)
	case Supply:
		bz, err = p.supplyOf(ctx, method, args)
	case TotalSupply:
		bz, err = p.totalSupply(ctx, method, args)
	default:
		bz, err = nil, errors.New("method not found")
	}

	return bz, err
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

func (p PrecompiledBank) send(ctx sdk.Context, stateDB vm.StateDB, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

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

	err = p.EmitSendEvent(ctx, stateDB, common.BytesToAddress(fromAddress.Bytes()), common.BytesToAddress(toAddress.Bytes()), coins)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p PrecompiledBank) totalSupply(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf(cmn.ErrInvalidNumberOfArgs, 1, len(args))
	}

	var input TotalSupplyInput
	if err := method.Inputs.Copy(&input, args); err != nil {
		return nil, fmt.Errorf("failed to copy args to struct: %w", err)
	}

	res, err := p.bk.TotalSupply(ctx, &banktypes.QueryTotalSupplyRequest{Pagination: &input.PageRequest})
	if err != nil {
		return nil, err
	}

	abiCoins := cmn.NewCoinsResponse(res.Supply)

	return method.Outputs.Pack(abiCoins, res.Pagination)
}
