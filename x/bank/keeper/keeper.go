package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/xpladev/xpla/x/bank/types"
)

var _ bankkeeper.Keeper = (*Keeper)(nil)

type Keeper struct {
	bankkeeper.BaseKeeper

	cdc codec.BinaryCodec

	bek BaseErc20Keeper
	bck BaseCw20Keeper

	ak banktypes.AccountKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak banktypes.AccountKeeper,
	blockedAddrs map[string]bool,
	authority string,
	logger log.Logger,
	ek types.EvmKeeper,
	wk types.WasmKeeper,
	wmk types.WasmMsgServer,
) Keeper {
	return Keeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(cdc, storeService, ak, blockedAddrs, authority, logger),
		cdc:        cdc,
		bek:        NewBaseErc20Keeper(ek),
		bck:        NewBaseCw20Keeper(wk, wmk),
		ak:         ak,
	}
}

func (k Keeper) GetBalance(goCtx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenType, address := types.ParseDenom(denom)
	switch tokenType {
	case types.Erc20:
		return k.bek.GetBalance(ctx, addr, address)
	case types.Cw20:
		return k.bck.GetBalance(ctx, addr, address)
	default:
		return k.BaseKeeper.GetBalance(ctx, addr, denom)
	}
}

func (k Keeper) GetSupply(goCtx context.Context, denom string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenType, address := types.ParseDenom(denom)
	switch tokenType {
	case types.Erc20:
		return k.bek.GetSupply(goCtx, address)
	case types.Cw20:
		return k.bck.GetSupply(goCtx, address)
	default:
		return k.BaseKeeper.GetSupply(ctx, denom)
	}
}

func (k Keeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	evmCoins := sdk.NewCoins()
	cw20Coins := sdk.NewCoins()
	cosmosCoins := sdk.NewCoins()

	for _, coin := range amt {
		tokenType, _ := types.ParseDenom(coin.Denom)
		switch tokenType {
		case types.Erc20:
			evmCoins = append(evmCoins, coin)
		case types.Cw20:
			cw20Coins = append(cw20Coins, coin)
		default:
			cosmosCoins = append(cosmosCoins, coin)
		}
	}

	if err := k.bek.SendCoins(ctx, fromAddr, toAddr, evmCoins); err != nil {
		return err
	}
	if err := k.bck.SendCoins(ctx, fromAddr, toAddr, cw20Coins); err != nil {
		return err
	}
	if err := k.BaseKeeper.SendCoins(ctx, fromAddr, toAddr, cosmosCoins); err != nil {
		return err
	}

	return nil
}

func (k Keeper) IsSendEnabledCoins(ctx context.Context, coins ...sdk.Coin) error {
	cosmosCoins := sdk.NewCoins()

	for _, coin := range coins {
		tokenType, _ := types.ParseDenom(coin.Denom)
		if tokenType == types.Cosmos {
			cosmosCoins = append(cosmosCoins, coin)
		}
	}
	return k.BaseKeeper.IsSendEnabledCoins(ctx, cosmosCoins...)
}

// SpendableCoin returns the balance of specific denomination of spendable coins
// for an account by address. If the account has no spendable coin, a zero Coin
// is returned.
// Copyed from cosmos-sdk/x/bank/keeper/view.go
func (k Keeper) SpendableCoin(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	balance := k.GetBalance(ctx, addr, denom)
	locked := k.LockedCoins(ctx, addr)
	return balance.SubAmount(locked.AmountOf(denom))
}
