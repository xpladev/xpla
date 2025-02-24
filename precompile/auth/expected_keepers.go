package auth

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) (acc sdk.AccountI)
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
}
