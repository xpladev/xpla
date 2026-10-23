package ics20

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/core/vm"

	pcommon "github.com/xpladev/xpla/precompile/common"
	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

// Precompile shares the active EVM state with xERC20 bank transfers.
type Precompile struct {
	*ics20precompile.Precompile
}

// NewPrecompile creates an ICS20 precompile that shares EVM state with xERC20 transfers.
func NewPrecompile(
	bankKeeper cmn.BankKeeper,
	stakingKeeper cmn.StakingKeeper,
	transferKeeper cmn.TransferKeeper,
	channelKeeper cmn.ChannelKeeper,
	erc20Keeper cmn.ERC20Keeper,
) Precompile {
	return Precompile{
		Precompile: ics20precompile.NewPrecompile(bankKeeper, stakingKeeper, transferKeeper, channelKeeper, erc20Keeper),
	}
}

func (p Precompile) Run(evm *vm.EVM, contract *vm.Contract, readonly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		ctx = xbanktypes.WithEVMStateDB(ctx, evm.StateDB.(*statedb.StateDB))

		initialGas := ctx.GasMeter().GasConsumed()
		bz, err := p.Execute(ctx, evm.StateDB, contract, readonly)
		if err != nil {
			pcommon.ChargeFailedExecutionGas(contract, ctx.GasMeter().GasConsumed()-initialGas)
		}
		return bz, err
	})
}
