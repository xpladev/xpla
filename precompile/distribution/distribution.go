package distribution

import (
	"context"
	"embed"
	"errors"
	"math/big"

	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"

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
			KvGasConfig:          storetypes.GasConfig{},
			TransientKVGasConfig: storetypes.GasConfig{},
		},
		dk: dk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledDistribution) Address() common.Address { return Address }

func (p PrecompiledDistribution) RequiredGas(input []byte) uint64 {
	// Implement the method as needed
	return 0
}

func (p PrecompiledDistribution) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
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

	switch MethodDistribution(abiMethod.Name) {
	case WithdrawDelegatorReward:
		return p.withdrawDelegatorReward(ctx, evm.Origin, abiMethod, args)
	default:
		return nil, errors.New("method not found")
	}
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
