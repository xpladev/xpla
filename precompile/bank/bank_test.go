package bank

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

type recordingBankKeeper struct {
	BankKeeper

	balanceCalls int
	supplyCalls  int
}

func (k *recordingBankKeeper) GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin {
	k.balanceCalls++
	return sdk.Coin{}
}

func (k *recordingBankKeeper) GetSupply(context.Context, string) sdk.Coin {
	k.supplyCalls++
	return sdk.Coin{}
}

func TestViewMethodsRejectMalformedReservedDenomBeforeKeeperCall(t *testing.T) {
	t.Run("balance", func(t *testing.T) {
		keeper := &recordingBankKeeper{}
		precompile := PrecompiledBank{bk: keeper}
		method := ABI.Methods[string(Balance)]

		_, err := precompile.balance(sdk.Context{}, &method, []interface{}{common.Address{}, "xerc20:invalid"})

		require.Error(t, err)
		require.Zero(t, keeper.balanceCalls)
	})

	t.Run("supply", func(t *testing.T) {
		keeper := &recordingBankKeeper{}
		precompile := PrecompiledBank{bk: keeper}
		method := ABI.Methods[string(Supply)]

		_, err := precompile.supplyOf(sdk.Context{}, &method, []interface{}{"xcw20:invalid"})

		require.Error(t, err)
		require.Zero(t, keeper.supplyCalls)
	})
}
