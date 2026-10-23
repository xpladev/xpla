package auth

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	xplaauthtypes "github.com/xpladev/xpla/x/auth/types"
)

type AccountKeeper interface {
	GetModuleAccount(ctx context.Context, moduleName string) sdk.ModuleAccountI
	ResolveAccountAddress(ctx context.Context, addr sdk.AccAddress, wasmKeeper xplaauthtypes.WasmKeeper) (sdk.AccAddress, error)
}
