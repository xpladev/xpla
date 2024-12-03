package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	types "github.com/xpladev/xpla/x/bank/types"
)

type BaseEvmKeeper struct {
	Erc20SendKeeper
}

func NewBaseErc20Keeper(ak banktypes.AccountKeeper, ek types.EvmKeeper) BaseEvmKeeper {
	erc20keeper := NewErc20Keeper(ek)
	return BaseEvmKeeper{
		Erc20SendKeeper: Erc20SendKeeper{
			Erc20ViewKeeper: Erc20ViewKeeper{erc20keeper: erc20keeper},
			erc20keeper:     erc20keeper,
		},
	}
}

type Erc20SendKeeper struct {
	Erc20ViewKeeper

	erc20keeper Erc20Keeper
}

// SendCoins implements keeper.SendKeeper.
func (k *Erc20SendKeeper) SendCoins(goCtx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	for _, coin := range amt {
		tokenType, address := types.ParseDenom(coin.Denom)
		if tokenType == types.Erc20 {
			contractAddress := common.HexToAddress(address)
			if err := k.erc20keeper.ExecuteTransfer(ctx, contractAddress, fromAddr, toAddr, coin.Amount.BigInt()); err != nil {
				return err
			}
		} else {
			return sdkerrors.ErrInvalidCoins.Wrapf("it should be erc20 token: %s", coin.String())
		}
	}

	return nil
}

type Erc20ViewKeeper struct {
	erc20keeper Erc20Keeper
}

// GetBalance implements keeper.ViewKeeper.
func (e *Erc20ViewKeeper) GetBalance(goCtx context.Context, addr sdk.AccAddress, hexErc20Address string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contractAddress := common.HexToAddress(hexErc20Address)

	amount, err := e.erc20keeper.QueryBalanceOf(ctx, contractAddress, addr)
	if err != nil {
		panic(err)
	}

	return sdk.NewCoin(types.ERC20+"/"+hexErc20Address, amount)

}
