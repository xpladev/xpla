package staking

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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	cmn "github.com/cosmos/evm/precompiles/common"

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
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		sk: sk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledStaking) Address() common.Address { return Address }

func (p PrecompiledStaking) RequiredGas(input []byte) uint64 {
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

func (p PrecompiledStaking) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	// Start the balance change handler before executing the precompile.
	p.GetBalanceHandler().BeforeBalanceChange(ctx)

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch MethodStaking(method.Name) {
	case Delegate:
		bz, err = p.delegate(ctx, evm.Origin, method, args)
	case BeginRedelegate:
		bz, err = p.beginRedelegate(ctx, evm.Origin, method, args)
	case Undelegate:
		bz, err = p.undelegate(ctx, evm.Origin, method, args)
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

func (p PrecompiledStaking) IsTransaction(method *abi.Method) bool {
	return false
}

func (p PrecompiledStaking) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("xpla evm extension", "staking")
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
