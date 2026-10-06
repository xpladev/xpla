package bank

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/common"
)

// exactBalanceBankKeeper is used only by the EVM balance-event handler, not by
// native bank transfers. Non-EVM addresses retain their exact bank balances and
// events, but must not be mirrored into an unrelated 20-byte StateDB account.
type exactBalanceBankKeeper struct {
	cmn.BankKeeper
}

func (k exactBalanceBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	return len(addr) != common.AddressLength || k.BankKeeper.BlockedAddr(addr)
}

// NewExactBalanceHandlerFactory returns a balance handler for stateful
// precompiles. It leaves the caller's BankKeeper unchanged.
func NewExactBalanceHandlerFactory(bk cmn.BankKeeper) *cmn.BalanceHandlerFactory {
	return cmn.NewBalanceHandlerFactory(exactBalanceBankKeeper{BankKeeper: bk})
}
