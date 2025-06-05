package staking

import (
	"embed"
	"errors"

	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"

	"github.com/xpladev/xpla/precompile/util"
)

var _ vm.PrecompiledContract = PrecompiledStaking{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IStaking.abi
	abiFS embed.FS
)

type PrecompiledStaking struct {
	cmn.Precompile
	sk StakingKeeper
}

func init() {
	var err error
	ABI, err = util.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledStaking(sk StakingKeeper) PrecompiledStaking {
	p := PrecompiledStaking{
		Precompile: cmn.Precompile{
			ABI:                  ABI,
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		sk: sk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledStaking) Address() common.Address { return Address }

func (p PrecompiledStaking) RequiredGas(input []byte) uint64 {
	// Implement the method as needed
	return 0
}

func (p PrecompiledStaking) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
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

	switch MethodStaking(abiMethod.Name) {
	case Delegate:
		return p.delegate(ctx, evm.Origin, abiMethod, args)
	case BeginRedelegate:
		return p.beginRedelegate(ctx, evm.Origin, abiMethod, args)
	case Undelegate:
		return p.undelegate(ctx, evm.Origin, abiMethod, args)
	default:
		return nil, errors.New("method not found")
	}
}

func (p PrecompiledStaking) delegate(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
	delegatorAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(delegatorAddress, sender); err != nil {
		return nil, err
	}

	validatorAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	coin, err := util.GetCoin(args[2])
	if err != nil {
		return nil, err
	}

	msg := stakingtypes.NewMsgDelegate(
		delegatorAddress.String(),
		sdk.ValAddress(validatorAddress.Bytes()).String(),
		coin,
	)

	_, err = p.sk.Delegate(ctx, msg)

	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(true)
}

func (p PrecompiledStaking) beginRedelegate(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
	delegatorAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(delegatorAddress, sender); err != nil {
		return nil, err
	}

	validatorSrcAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	validatorDstAddress, err := util.GetAccAddress(args[2])
	if err != nil {
		return nil, err
	}

	coin, err := util.GetCoin(args[3])
	if err != nil {
		return nil, err
	}

	msg := stakingtypes.NewMsgBeginRedelegate(
		delegatorAddress.String(),
		sdk.ValAddress(validatorSrcAddress.Bytes()).String(),
		sdk.ValAddress(validatorDstAddress.Bytes()).String(),
		coin,
	)

	res, err := p.sk.BeginRedelegate(ctx.Context(), msg)

	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.CompletionTime.Unix())
}

func (p PrecompiledStaking) undelegate(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
	delegatorAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(delegatorAddress, sender); err != nil {
		return nil, err
	}

	validatorAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	coin, err := util.GetCoin(args[2])
	if err != nil {
		return nil, err
	}

	msg := stakingtypes.NewMsgUndelegate(
		delegatorAddress.String(),
		sdk.ValAddress(validatorAddress.Bytes()).String(),
		coin,
	)

	res, err := p.sk.Undelegate(ctx.Context(), msg)

	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.CompletionTime.Unix())
}
