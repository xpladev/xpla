package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	types "github.com/xpladev/xpla/x/bank/types"
)

type BaseCw20Keeper struct {
	Cw20SendKeeper
}

func NewBaseCw20Keeper(wk types.WasmKeeper, wmk types.WasmMsgServer) BaseCw20Keeper {
	cw20keeper := NewCw20Keeper(wk, wmk)
	return BaseCw20Keeper{
		Cw20SendKeeper: Cw20SendKeeper{
			Cw20ViewKeeper: Cw20ViewKeeper{cw20keeper: cw20keeper},
			cw20keeper:     cw20keeper,
		},
	}
}

func (k BaseCw20Keeper) GetSupply(goCtx context.Context, contractAddress string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)

	tokenContractAddress := sdk.MustAccAddressFromBech32(contractAddress)
	tokenInfo, err := k.cw20keeper.QueryTokenInfo(ctx, tokenContractAddress)
	if err != nil {
		return types.NewCw20Coin(contractAddress, sdkmath.ZeroInt())
	}

	totalSupply, ok := sdkmath.NewIntFromString(string(tokenInfo.TotalSupply))
	if !ok {
		return types.NewCw20Coin(contractAddress, sdkmath.ZeroInt())
	}

	return types.NewCw20Coin(contractAddress, totalSupply)
}

type Cw20SendKeeper struct {
	Cw20ViewKeeper

	cw20keeper Cw20Keeper
}

// SendCoins implements keeper.SendKeeper.
func (k Cw20SendKeeper) SendCoins(goCtx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	ctx := sdk.UnwrapSDKContext(goCtx)

	for _, coin := range amt {
		tokenType, address := types.ParseDenom(coin.Denom)
		if tokenType == types.Cw20 {
			contractAddress, err := sdk.AccAddressFromBech32(address)
			if err != nil {
				return err
			}
			transferMsg := &types.ExecuteMsg_Transfer{
				Recipient: toAddr.String(),
				Amount:    types.Uint128(coin.Amount.String()),
			}
			if _, err := k.cw20keeper.ExecuteTransfer(ctx, fromAddr, contractAddress, transferMsg); err != nil {
				return err
			}
		} else {
			return sdkerrors.ErrInvalidCoins.Wrapf("it should be cw20 token: %s", coin.String())
		}
	}

	return nil
}

type Cw20ViewKeeper struct {
	cw20keeper Cw20Keeper
}

// GetBalance implements keeper.ViewKeeper.
func (e Cw20ViewKeeper) GetBalance(goCtx context.Context, addr sdk.AccAddress, cw20Address string) sdk.Coin {
	ctx := sdk.UnwrapSDKContext(goCtx)
	contractAddress := sdk.MustAccAddressFromBech32(cw20Address)

	balanceReq := &types.QueryMsg_Balance{
		Address: addr.String(),
	}
	balanceResp, err := e.cw20keeper.QueryBalance(ctx, contractAddress, balanceReq)
	if err != nil {
		return types.NewCw20Coin(cw20Address, sdkmath.ZeroInt())
	}

	amount, ok := sdkmath.NewIntFromString(string(balanceResp.Balance))
	if !ok {
		return types.NewCw20Coin(cw20Address, sdkmath.ZeroInt())
	}

	return types.NewCw20Coin(cw20Address, amount)
}
