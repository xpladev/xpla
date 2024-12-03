package keeper

import (
	"context"

	"github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type msgServer struct {
	banktypes.MsgServer
	Keeper
}

var _ banktypes.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the bank MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) banktypes.MsgServer {
	return &msgServer{
		MsgServer: bankkeeper.NewMsgServerImpl(keeper.BaseKeeper),
		Keeper:    keeper}
}

// Send implements types.MsgServer.
func (k msgServer) Send(goCtx context.Context, msg *banktypes.MsgSend) (*banktypes.MsgSendResponse, error) {
	var (
		from, to []byte
		err      error
	)

	from, err = k.ak.AddressCodec().StringToBytes(msg.FromAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid from address: %s", err)
	}
	to, err = k.ak.AddressCodec().StringToBytes(msg.ToAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid to address: %s", err)
	}

	if !msg.Amount.IsValid() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	if !msg.Amount.IsAllPositive() {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidCoins, msg.Amount.String())
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := k.IsSendEnabledCoins(ctx, msg.Amount...); err != nil {
		return nil, err
	}

	if k.BlockedAddr(to) {
		return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", msg.ToAddress)
	}

	err = k.SendCoins(ctx, from, to, msg.Amount)
	if err != nil {
		return nil, err
	}

	defer func() {
		for _, a := range msg.Amount {
			if a.Amount.IsInt64() {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "send"},
					float32(a.Amount.Int64()),
					[]metrics.Label{telemetry.NewLabel("denom", a.Denom)},
				)
			}
		}
	}()

	return &banktypes.MsgSendResponse{}, nil
}

// MultiSend implements types.MsgServer.
func (k msgServer) MultiSend(goCtx context.Context, msg *banktypes.MsgMultiSend) (*banktypes.MsgMultiSendResponse, error) {
	if len(msg.Inputs) == 0 {
		return nil, banktypes.ErrNoInputs
	}

	if len(msg.Inputs) != 1 {
		return nil, banktypes.ErrMultipleSenders
	}

	if len(msg.Outputs) == 0 {
		return nil, banktypes.ErrNoOutputs
	}

	if err := banktypes.ValidateInputOutputs(msg.Inputs[0], msg.Outputs); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	// NOTE: totalIn == totalOut should already have been checked
	for _, in := range msg.Inputs {
		if err := k.IsSendEnabledCoins(ctx, in.Coins...); err != nil {
			return nil, err
		}
	}

	for _, out := range msg.Outputs {
		accAddr, err := k.ak.AddressCodec().StringToBytes(out.Address)
		if err != nil {
			return nil, err
		}

		if k.BlockedAddr(accAddr) {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "%s is not allowed to receive funds", out.Address)
		}
	}

	err := k.InputOutputCoins(ctx, msg.Inputs[0], msg.Outputs)
	if err != nil {
		return nil, err
	}

	return &banktypes.MsgMultiSendResponse{}, nil
}
