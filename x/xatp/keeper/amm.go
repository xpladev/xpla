package keeper

import (
	"encoding/json"
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

const (
	executeTransferCw20 = `{
		"transfer": {
			"recipient":  "%s",
			"amount": "%s"
		}
	}`

	queryPair         = `{"pair":{}}`
	queryTokenInfo    = `{"token_info":{}}`
	queryPool         = `{"pool":{}}`
	queryTokenBalance = `{"balance":{"address":"%s"}}`
)

func (k Keeper) Pair(ctx sdk.Context, pairContract string) (*types.Pair, error) {
	pairAddress, err := sdk.AccAddressFromBech32(pairContract)
	if err != nil {
		return nil, err
	}

	res, err := k.viewKeeper.QuerySmart(ctx, pairAddress, []byte(queryPair))
	if err != nil {
		return nil, err
	}

	pair := &types.Pair{}
	err = json.Unmarshal(res, pair)
	if err != nil {
		return nil, err
	}

	return pair, nil
}

func (k Keeper) TokenInfo(ctx sdk.Context, tokenContract string) (*types.Token, error) {
	tokenAddress, err := sdk.AccAddressFromBech32(tokenContract)
	if err != nil {
		return nil, err
	}

	res, err := k.viewKeeper.QuerySmart(ctx, tokenAddress, []byte(queryTokenInfo))
	if err != nil {
		return nil, err
	}

	token := &types.Token{}
	err = json.Unmarshal(res, token)
	if err != nil {
		return nil, err
	}

	return token, nil
}

func (k Keeper) Pool(ctx sdk.Context, pairContract string) (*types.Pool, error) {
	pairAddress, err := sdk.AccAddressFromBech32(pairContract)
	if err != nil {
		return nil, err
	}

	res, err := k.viewKeeper.QuerySmart(ctx, pairAddress, []byte(queryPool))
	if err != nil {
		return nil, err
	}

	pool := &types.Pool{}
	err = json.Unmarshal(res, pool)
	if err != nil {
		return nil, err
	}

	return pool, nil
}

func (k Keeper) TransferCw20(ctx sdk.Context, sender sdk.AccAddress, token string, amount string, recipient string) error {
	tokenAddress, err := sdk.AccAddressFromBech32(token)
	if err != nil {
		return err
	}

	msg := fmt.Sprintf(executeTransferCw20, recipient, amount)

	_, err = k.contractKeeper.Execute(
		ctx,
		tokenAddress,
		sender,
		wasmtypes.RawContractMessage(msg),
		sdk.NewCoins(),
	)

	if err != nil {
		return err
	}

	return nil
}

func (k Keeper) TokenBalance(ctx sdk.Context, tokenContract string, account sdk.AccAddress) (*types.Cw20Balance, error) {
	tokenAddress, err := sdk.AccAddressFromBech32(tokenContract)
	if err != nil {
		return nil, err
	}

	params := fmt.Sprintf(queryTokenBalance, account.String())
	res, err := k.viewKeeper.QuerySmart(ctx, tokenAddress, []byte(params))
	if err != nil {
		return nil, err
	}

	balance := &types.Cw20Balance{}
	err = json.Unmarshal(res, balance)
	if err != nil {
		return nil, err
	}

	return balance, nil
}
