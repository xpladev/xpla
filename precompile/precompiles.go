package precompile

import (
	"github.com/ethereum/go-ethereum/core/vm"

	pbank "github.com/xpladev/xpla/precompile/bank"
	pdistribution "github.com/xpladev/xpla/precompile/distribution"
	pstaking "github.com/xpladev/xpla/precompile/staking"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
)

func RegistPrecompiledContract(ak pwasm.AccountKeeper, bk pbank.BankKeeper, sk pstaking.StakingKeeper, dk pdistribution.DistributionKeeper, wms pwasm.WasmMsgServer, wk pwasm.WasmKeeper) {
	vm.PrecompiledContractsBerlin[pbank.Address] = pbank.NewPrecompiledBank(bk)
	vm.PrecompiledContractsBerlin[pstaking.Address] = pstaking.NewPrecompiledStaking(sk)
	vm.PrecompiledContractsBerlin[pdistribution.Address] = pdistribution.NewPrecompiledDistribution(dk)
	vm.PrecompiledContractsBerlin[pwasm.Address] = pwasm.NewPrecompiledWasm(ak, wms, wk)
}
