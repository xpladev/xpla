package bank

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type BankKeeper interface {
	GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin
	GetSupply(context.Context, string) sdk.Coin
	SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error
}
