package distribution

import (
	"context"
	"embed"
	"errors"
	"math/big"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/xpladev/xpla/precompile/util"
)

var _ vm.PrecompiledContract = PrecompiledDistribution{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IDistribution.abi
	abiFS embed.FS
)

type PrecompiledDistribution struct {
	cmn.Precompile
	dk DistributionKeeper
}

func init() {
	var err error
	ABI, err = util.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledDistribution(dk DistributionKeeper) PrecompiledDistribution {
	p := PrecompiledDistribution{
		Precompile: cmn.Precompile{
			ABI:                  ABI,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		dk: dk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledDistribution) Address() common.Address { return Address }

func (p PrecompiledDistribution) RequiredGas(input []byte) uint64 {
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

func (p PrecompiledDistribution) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	// Start the balance change handler before executing the precompile.
	p.GetBalanceHandler().BeforeBalanceChange(ctx)

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch MethodDistribution(method.Name) {
	case WithdrawDelegatorReward:
		bz, err = p.withdrawDelegatorReward(ctx, evm.Origin, method, args)
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

func (p PrecompiledDistribution) IsTransaction(method *abi.Method) bool {
	return false
}

func (p PrecompiledDistribution) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("xpla evm extension", "distribution")
}

func (p PrecompiledDistribution) withdrawDelegatorReward(ctx context.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
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

	msg := distributiontypes.NewMsgWithdrawDelegatorReward(delegatorAddress.String(), sdk.ValAddress(validatorAddress.Bytes()).String())

	res, err := p.dk.WithdrawDelegatorReward(ctx, msg)
	if err != nil {
		return nil, err
	}

	amount := big.NewInt(0)
	if !res.Amount.IsZero() {
		amount = res.Amount[0].Amount.BigInt()
	}

	return method.Outputs.Pack(amount)
}
