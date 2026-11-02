package keepers

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

var _ wasmkeeper.Messenger = wasmEVMStateDBAtomicMessenger{}

// wasmEVMStateDBAtomicMessenger extends Wasmd's submessage cache boundary to
// the shared EVM StateDB injected by the Wasm precompile. Wasmd continues to
// own the incoming SDK cache; this decorator gives the dispatch a private
// native scope whose SDK and EVM changes write or roll back together.
type wasmEVMStateDBAtomicMessenger struct {
	next wasmkeeper.Messenger
}

func newWasmEVMStateDBAtomicMessenger(next wasmkeeper.Messenger) wasmkeeper.Messenger {
	return wasmEVMStateDBAtomicMessenger{next: next}
}

func (m wasmEVMStateDBAtomicMessenger) DispatchMsg(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contractIBCPortID string,
	msg wasmvmtypes.CosmosMsg,
) (events []sdk.Event, data [][]byte, msgResponses [][]*codectypes.Any, err error) {
	stateDB, ok := xbanktypes.EVMStateDBFromContext(ctx)
	if !ok {
		return m.next.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	}

	scopeCtx, scope, err := stateDB.NewNativeScope(ctx)
	if err != nil {
		return nil, nil, nil, err
	}
	// Rollback also runs for panics, including ErrorOutOfGas. Gas meters are
	// deliberately outside the scope rollback boundary.
	defer scope.Rollback()

	events, data, msgResponses, err = m.next.DispatchMsg(scopeCtx, contractAddr, contractIBCPortID, msg)
	if err != nil {
		return events, data, msgResponses, err
	}
	if err = scope.Write(); err != nil {
		return events, data, msgResponses, err
	}
	return events, data, msgResponses, err
}
