package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/xpladev/xpla/x/auth/types"
)

// AccountKeeper adds query address resolution while inheriting exact account
// identity and storage operations from the SDK keeper.
type AccountKeeper struct {
	authkeeper.AccountKeeper
}

func NewAccountKeeper(base authkeeper.AccountKeeper) AccountKeeper {
	return AccountKeeper{AccountKeeper: base}
}

// ResolveAccountAddress returns nil when neither an exact account nor an alias exists.
func (k AccountKeeper) ResolveAccountAddress(ctx context.Context, addr sdk.AccAddress, wasmKeeper types.WasmKeeper) (sdk.AccAddress, error) {
	if account := k.GetAccount(ctx, addr); account != nil {
		return account.GetAddress(), nil
	}
	if len(addr) != 20 {
		return nil, nil
	}
	contractAddr, found, err := wasmKeeper.ResolveWasmAlias(ctx, addr)
	if err != nil || !found {
		return nil, err
	}
	// Wasm resolution already validates the target account. Both consumers need
	// only its address; SDK RPC queries load the account when building responses.
	return contractAddr, nil
}
