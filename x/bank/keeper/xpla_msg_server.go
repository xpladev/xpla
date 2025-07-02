package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/xpladev/xpla/x/bank/types"
)

type xplaMsgServer struct {
	Keeper
}

var _ types.MsgServer = xplaMsgServer{}

// NewXplaMsgServerImpl returns an implementation of the XPLA bank MsgServer interface
// for the provided Keeper.
func NewXplaMsgServerImpl(keeper Keeper) types.MsgServer {
	return &xplaMsgServer{
		Keeper: keeper}
}

// Burn implements XPLA bank MsgServer for burning coins.
func (k xplaMsgServer) Burn(goCtx context.Context, req *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	// Validate amount
	if !req.Amount.IsValid() || !req.Amount.IsAllPositive() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, req.Amount.String())
	}

	// Burn the coins from gov module account
	err := k.BurnCoins(ctx, govtypes.ModuleName, req.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to burn coins")
	}

	return &types.MsgBurnResponse{}, nil
}
