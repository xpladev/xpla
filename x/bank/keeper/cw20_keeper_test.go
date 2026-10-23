package keeper

import (
	"bytes"
	"context"
	"errors"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

type recordingWasmQueryKeeper struct {
	calls int
	err   error
}

func (k *recordingWasmQueryKeeper) QuerySmart(context.Context, sdk.AccAddress, []byte) ([]byte, error) {
	k.calls++
	return nil, k.err
}

func TestCw20ViewKeepersRejectInvalidContractAddressWithoutQuery(t *testing.T) {
	t.Run("balance", func(t *testing.T) {
		wasmKeeper := &recordingWasmQueryKeeper{}
		keeper := NewBaseCw20Keeper(wasmKeeper, nil)

		var balance sdk.Coin
		require.NotPanics(t, func() {
			balance = keeper.GetBalance(context.Background(), sdk.AccAddress{1}, "invalid")
		})

		require.True(t, balance.Amount.IsZero())
		require.Zero(t, wasmKeeper.calls)
	})

	t.Run("supply", func(t *testing.T) {
		wasmKeeper := &recordingWasmQueryKeeper{}
		keeper := NewBaseCw20Keeper(wasmKeeper, nil)

		var supply sdk.Coin
		require.NotPanics(t, func() {
			supply = keeper.GetSupply(context.Background(), "invalid")
		})

		require.True(t, supply.Amount.IsZero())
		require.Zero(t, wasmKeeper.calls)
	})
}

func TestCw20ViewKeepersReturnZeroOnQueryError(t *testing.T) {
	contractAddress := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	ctx := sdk.Context{}.WithContext(context.Background())

	t.Run("balance", func(t *testing.T) {
		wasmKeeper := &recordingWasmQueryKeeper{err: errors.New("query failed")}
		keeper := NewBaseCw20Keeper(wasmKeeper, nil)

		balance := keeper.GetBalance(ctx, sdk.AccAddress{2}, contractAddress)

		require.True(t, balance.Amount.IsZero())
		require.Equal(t, 1, wasmKeeper.calls)
	})

	t.Run("supply", func(t *testing.T) {
		wasmKeeper := &recordingWasmQueryKeeper{err: errors.New("query failed")}
		keeper := NewBaseCw20Keeper(wasmKeeper, nil)

		supply := keeper.GetSupply(ctx, contractAddress)

		require.True(t, supply.Amount.IsZero())
		require.Equal(t, 1, wasmKeeper.calls)
	})
}
