package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// WasmKeeper supplies the alias lookup used by auth queries, including target
// account validation. Alias storage and lifecycle remain owned by Wasm.
type WasmKeeper interface {
	ResolveWasmAlias(context.Context, sdk.AccAddress) (sdk.AccAddress, bool, error)
}
