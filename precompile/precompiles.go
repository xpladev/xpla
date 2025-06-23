package precompile

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	pauth "github.com/xpladev/xpla/precompile/auth"
	pbank "github.com/xpladev/xpla/precompile/bank"
	pdistribution "github.com/xpladev/xpla/precompile/distribution"
	pstaking "github.com/xpladev/xpla/precompile/staking"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
)

var PrecompiledAddressesXpla = []common.Address{
	pbank.Address, pstaking.Address, pdistribution.Address, pwasm.Address, pauth.Address,
}

func RegistPrecompiledContract(ak pwasm.AccountKeeper, bk pbank.BankKeeper, sk pstaking.StakingKeeper, dk pdistribution.DistributionKeeper, wms pwasm.WasmMsgServer, wk pwasm.WasmKeeper, authAk pauth.AccountKeeper) {
	vm.PrecompiledContractsBerlin[pbank.Address] = pbank.NewPrecompiledBank(bk)
	vm.PrecompiledContractsBerlin[pstaking.Address] = pstaking.NewPrecompiledStaking(sk)
	vm.PrecompiledContractsBerlin[pdistribution.Address] = pdistribution.NewPrecompiledDistribution(dk)
	vm.PrecompiledContractsBerlin[pwasm.Address] = pwasm.NewPrecompiledWasm(ak, wms, wk)
	vm.PrecompiledContractsBerlin[pauth.Address] = pauth.NewPrecompiledAuth(authAk)
}
