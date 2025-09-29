package bank

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type BankKeeper interface {
	IterateAccountBalances(context.Context, sdk.AccAddress, func(coin sdk.Coin) bool)
	IterateTotalSupply(context.Context, func(coin sdk.Coin) bool)
	GetSupply(context.Context, string) sdk.Coin
	GetDenomMetaData(context.Context, string) (banktypes.Metadata, bool)
	SetDenomMetaData(context.Context, banktypes.Metadata)
	GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin
	SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error
	SpendableCoin(context.Context, sdk.AccAddress, string) sdk.Coin
	TotalSupply(context.Context, *banktypes.QueryTotalSupplyRequest) (*banktypes.QueryTotalSupplyResponse, error)
	BlockedAddr(sdk.AccAddress) bool
}
