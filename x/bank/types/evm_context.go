package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/evm/x/vm/statedb"
)

type evmStateDBContextKey struct{}

func WithEVMStateDB(ctx sdk.Context, stateDB *statedb.StateDB) sdk.Context {
	if stateDB == nil {
		return ctx
	}

	return ctx.WithValue(evmStateDBContextKey{}, stateDB)
}

func EVMStateDBFromContext(ctx sdk.Context) (*statedb.StateDB, bool) {
	stateDB, ok := ctx.Value(evmStateDBContextKey{}).(*statedb.StateDB)
	if !ok || stateDB == nil {
		return nil, false
	}

	return stateDB, true
}
