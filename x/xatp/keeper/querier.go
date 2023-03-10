package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func NewQuerier(k Keeper, legacyQuerierCdc *codec.LegacyAmino) sdk.Querier {
	return func(ctx sdk.Context, path []string, req abci.RequestQuery) ([]byte, error) {
		switch path[0] {
		case types.QueryParams:
			return queryParams(ctx, path[1:], req, k, legacyQuerierCdc)
		case types.QueryXatps:
			return queryXatps(ctx, req, k, legacyQuerierCdc)
		case types.QueryXatp:
			return queryXatp(ctx, req, k, legacyQuerierCdc)
		case types.QueryXatpPool:
			return queryXatpPool(ctx, req, k, legacyQuerierCdc)
		default:
			return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unknown query path: %s", path[0])
		}
	}
}

func queryParams(ctx sdk.Context, _ []string, _ abci.RequestQuery, k Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	params := k.GetParams(ctx)

	res, err := codec.MarshalJSONIndent(legacyQuerierCdc, params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryXatps(ctx sdk.Context, req abci.RequestQuery, k Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	xatps := k.GetAllXatps(ctx)

	res, err := codec.MarshalJSONIndent(legacyQuerierCdc, xatps)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryXatp(ctx sdk.Context, req abci.RequestQuery, k Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	var params types.QueryXatpRequest

	err := legacyQuerierCdc.UnmarshalJSON(req.Data, &params)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONUnmarshal, err.Error())
	}

	xatp, found := k.GetXatp(ctx, params.Denom)
	if !found {
		return nil, sdkerrors.Wrap(sdkerrors.ErrNotFound, params.Denom)
	}

	res, err := codec.MarshalJSONIndent(legacyQuerierCdc, xatp)
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return res, nil
}

func queryXatpPool(ctx sdk.Context, _ abci.RequestQuery, k Keeper, legacyQuerierCdc *codec.LegacyAmino) ([]byte, error) {
	balances := k.bankKeeper.GetAllBalances(ctx, k.GetXatpPayerAccount())

	xatps := k.GetAllXatps(ctx)
	xatpAccount := k.GetXatpPayerAccount()
	for _, xatp := range xatps {
		balance := sdk.ZeroInt()
		res, err := k.TokenBalance(ctx, xatp.Token, xatpAccount)
		if err == nil {
			var ok bool
			balance, ok = sdk.NewIntFromString(res.Balance)
			if !ok {
				balance = sdk.ZeroInt()
			}
		}
		balances = balances.Add(sdk.NewCoin(xatp.Denom, balance))
	}

	bz, err := legacyQuerierCdc.MarshalJSON(sdk.NewDecCoinsFromCoins(balances...))
	if err != nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrJSONMarshal, err.Error())
	}

	return bz, nil
}
