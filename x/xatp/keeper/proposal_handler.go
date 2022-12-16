package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func NewXatpProposalHandler(k Keeper) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		switch c := content.(type) {
		case *types.RegisterXatpProposal:
			return handlerRegisterXatpProposal(ctx, k, c)
		case *types.UnregisterXatpProposal:
			return handlerUnregisterXatpProposal(ctx, k, c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized xatp proposal content type: %T", c)
		}
	}
}

func handlerRegisterXatpProposal(ctx sdk.Context, k Keeper, p *types.RegisterXatpProposal) error {
	token, err := k.TokenInfo(ctx, p.Xatp.Token)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid token address")
	}

	if token.Symbol != p.Xatp.Denom {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "token denom")
	}

	if token.Decimals != int(p.Xatp.Decimals) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid token decimals")
	}

	pair, err := k.Pair(ctx, p.Xatp.Pair)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid pair address")
	}

	tokenInfo, tokenDecimals, err := pair.Xatp()
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, err.Error())
	}

	if tokenInfo.ContractAddr != p.Xatp.Token {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid pair address")
	}

	if tokenDecimals != int(p.Xatp.Decimals) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid token decimals")
	}

	k.SetXatp(ctx, *p.Xatp)

	return nil
}

func handlerUnregisterXatpProposal(ctx sdk.Context, k Keeper, p *types.UnregisterXatpProposal) error {
	k.DeleteXatp(ctx, p.Denom)

	return nil
}
