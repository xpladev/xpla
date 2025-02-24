package auth

import (
	"embed"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/ethermint/x/evm/statedb"

	"github.com/xpladev/xpla/precompile/util"
)

var _ vm.PrecompiledContract = PrecompiledAuth{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IAuth.abi
	abiFS embed.FS
)

type PrecompiledAuth struct {
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
	return PrecompiledAuth{ak}
}

func (p PrecompiledAuth) RequiredGas(input []byte) uint64 {
	// Implement the method as needed
	return 0
}

func (p PrecompiledAuth) Run(evm *vm.EVM, input []byte) ([]byte, error) {
	method, argsBz := util.SplitInput(input)

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
	case AssociatedAddress:
		return p.associatedAddress(ctx, abiMethod, args)
	default:
		return nil, errors.New("method not found")
	}
}

func (p PrecompiledAuth) associatedAddress(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
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
		// should be address type
		strAddress = address.String()
	}

	return method.Outputs.Pack([]byte(strAddress))
}
