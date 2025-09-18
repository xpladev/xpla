package bank

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BankKeeper interface {
	GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin
	GetSupply(context.Context, string) sdk.Coin
	SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error
	TotalSupply(context.Context, *banktypes.QueryTotalSupplyRequest) (*banktypes.QueryTotalSupplyResponse, error)
}
