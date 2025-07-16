package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/xpladev/xpla/x/burn/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the burn MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{
		Keeper: keeper}
}

// Burn implements burn MsgServer for burning coins.
func (k msgServer) Burn(goCtx context.Context, req *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if k.GetAuthority() != req.Authority {
		return nil, errorsmod.Wrapf(govtypes.ErrInvalidSigner, "invalid authority; expected %s, got %s", k.GetAuthority(), req.Authority)
	}

	// Validate amount
	if !req.Amount.IsValid() || !req.Amount.IsAllPositive() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, req.Amount.String())
	}

	// Burn the coins from gov module account
	err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, req.Amount)
	if err != nil {
		return nil, errorsmod.Wrap(err, "failed to burn coins")
	}

	return &types.MsgBurnResponse{}, nil
}
