package types

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/vm/statedb"
)

func TestEVMStateDBContext(t *testing.T) {
	ctx := sdk.Context{}.WithContext(context.Background())

	stateDB, ok := EVMStateDBFromContext(ctx)
	require.False(t, ok)
	require.Nil(t, stateDB)

	expected := &statedb.StateDB{}
	ctx = WithEVMStateDB(ctx, expected)

	stateDB, ok = EVMStateDBFromContext(ctx)
	require.True(t, ok)
	require.Same(t, expected, stateDB)
}

func TestWithEVMStateDBIgnoresNil(t *testing.T) {
	ctx := sdk.Context{}.WithContext(context.Background())
	ctx = WithEVMStateDB(ctx, nil)

	stateDB, ok := EVMStateDBFromContext(ctx)
	require.False(t, ok)
	require.Nil(t, stateDB)
}
