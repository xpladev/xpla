package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/xpladev/xpla/x/bank/types"
)

var _ bankkeeper.Keeper = (*Keeper)(nil)

type Keeper struct {
	bankkeeper.BaseKeeper

	bek BaseEvmKeeper

	ak banktypes.AccountKeeper
	ek types.EvmKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeService store.KVStoreService,
	ak banktypes.AccountKeeper,
	blockedAddrs map[string]bool,

	authority string,
	logger log.Logger,
	ek types.EvmKeeper,
) Keeper {
	return Keeper{
		BaseKeeper: bankkeeper.NewBaseKeeper(cdc, storeService, ak, blockedAddrs, authority, logger),
		bek:        NewBaseErc20Keeper(ak, ek),
		ak:         ak,
		ek:         ek,
	}
}

func (k *Keeper) SetEvmKeeper(ek types.EvmKeeper) {
	k.ek = ek
	k.bek = NewBaseErc20Keeper(k.ak, ek)
}
func (k Keeper) GetBalance(goCtx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenType, address := types.ParseDenom(denom)
	if tokenType == types.Erc20 {
		return k.bek.GetBalance(ctx, addr, address)
	}

	return k.BaseKeeper.GetBalance(ctx, addr, denom)
}

func (k Keeper) GetSupply(goCtx context.Context, denom string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenType, address := types.ParseDenom(denom)
	if tokenType == types.Erc20 {
		tokenContractAddress := common.HexToAddress(address)
		totalSupply, err := k.bek.erc20keeper.QueryTotalSupply(ctx, tokenContractAddress)
		if err != nil {
			panic(err)
		}

		return sdk.NewCoin(denom, totalSupply)
	}

	return k.BaseKeeper.GetSupply(ctx, denom)
}

func (k Keeper) SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	evmCoins := sdk.NewCoins()
	cosmosCoins := sdk.NewCoins()

	for _, coin := range amt {
		tokenType, _ := types.ParseDenom(coin.Denom)
		if tokenType == types.Erc20 {
			evmCoins = append(evmCoins, coin)
		} else {
			cosmosCoins = append(cosmosCoins, coin)
		}
	}

	if err := k.bek.SendCoins(ctx, fromAddr, toAddr, evmCoins); err != nil {
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

func (k Keeper) InputOutputCoins(ctx context.Context, input banktypes.Input, outputs []banktypes.Output) error {
	inputAddress, err := sdk.AccAddressFromBech32(input.Address)
	if err != nil {
		return err
	}

	outputsEvm := []banktypes.Output{}
	outputsCosmos := []banktypes.Output{}
	for _, output := range outputs {
		outputEvm := banktypes.Output{
			Address: output.Address,
			Coins:   sdk.NewCoins(),
		}

		outputCosmos := banktypes.Output{
			Address: output.Address,
			Coins:   sdk.NewCoins(),
		}
		for _, coin := range output.Coins {
			tokenType, _ := types.ParseDenom(coin.Denom)
			if tokenType == types.Erc20 {
				outputEvm.Coins = append(outputEvm.Coins, coin)
			} else {
				outputCosmos.Coins = append(outputCosmos.Coins, coin)
			}
		}

		if len(outputEvm.Coins) > 0 {
			outputsEvm = append(outputsEvm, outputEvm)
		}
		if len(outputCosmos.Coins) > 0 {
			outputsCosmos = append(outputsCosmos, outputCosmos)
		}
	}

	if len(outputsEvm) > 0 {
		for _, outputEvm := range outputsEvm {
			outputAddress, err := sdk.AccAddressFromBech32(outputEvm.Address)
			if err != nil {
				return err
			}

			if err := k.bek.SendCoins(ctx, inputAddress, outputAddress, outputEvm.Coins); err != nil {
				return err
			}
		}
	}

	if len(outputsCosmos) > 0 {
		if err := k.BaseKeeper.InputOutputCoins(ctx, input, outputsCosmos); err != nil {
			return err
		}
	}

	return nil
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
