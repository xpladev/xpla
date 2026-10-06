package bank

import (
	"bytes"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestExactBalanceBankKeeperExcludesNonEVMAddresses(t *testing.T) {
	for _, length := range []int{0, 19, 21, 32} {
		bank := &balanceHandlerBankStub{}
		adapter := exactBalanceBankKeeper{BankKeeper: bank}
		require.True(t, adapter.BlockedAddr(sdk.AccAddress(bytes.Repeat([]byte{1}, length))))
		require.Zero(t, bank.calls, "non-EVM addresses must not reach the EVM mirror")
	}
}

func TestExactBalanceBankKeeperPreservesTwentyByteBlocking(t *testing.T) {
	address := sdk.AccAddress(bytes.Repeat([]byte{1}, common.AddressLength))
	for _, blocked := range []bool{false, true} {
		bank := &balanceHandlerBankStub{blocked: blocked}
		adapter := exactBalanceBankKeeper{BankKeeper: bank}
		require.Equal(t, blocked, adapter.BlockedAddr(address))
		require.Equal(t, 1, bank.calls)
		require.Equal(t, address, bank.address)
	}
}

type balanceHandlerBankStub struct {
	cmn.BankKeeper
	blocked bool
	calls   int
	address sdk.AccAddress
}

func (k *balanceHandlerBankStub) BlockedAddr(address sdk.AccAddress) bool {
	k.calls++
	k.address = address
	return k.blocked
}
