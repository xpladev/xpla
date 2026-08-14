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
// own SDK store and event rollback through its CacheContext; this decorator
// only rolls back EVM journaled state and logs.
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

	snapshot := stateDB.Snapshot()
	succeeded := false

	// A panic, including ErrorOutOfGas, runs this defer and then propagates
	// unchanged. Gas meters are intentionally outside the rollback boundary.
	defer func() {
		if !succeeded {
			stateDB.RevertToSnapshot(snapshot)
		}
	}()

	events, data, msgResponses, err = m.next.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	succeeded = err == nil
	return events, data, msgResponses, err
}
