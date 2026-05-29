package keeper

import (
	"context"
	"time"

	"github.com/hashicorp/go-metrics"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

var _ stakingtypes.MsgServer = msgServer{}

type msgServer struct {
	stakingtypes.MsgServer

	keeper *Keeper
}

// NewMsgServerImpl returns a staking msg server that preserves XPLA's dust-share
// unbonding behavior while delegating the rest of the staking service to the SDK.
func NewMsgServerImpl(k *Keeper) stakingtypes.MsgServer {
	return msgServer{
		MsgServer: stakingkeeper.NewMsgServerImpl(k.Keeper),
		keeper:    k,
	}
}

// ValidateUnbondAmount keeps the previous XPLA tolerance for token-to-share
// conversion so a near-full token undelegation can clear the remaining shares.
func (k Keeper) ValidateUnbondAmount(
	ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, amt math.Int,
) (shares math.LegacyDec, err error) {
	validator, err := k.GetValidator(ctx, valAddr)
	if err != nil {
		return shares, err
	}

	del, err := k.GetDelegation(ctx, delAddr, valAddr)
	if err != nil {
		return shares, err
	}

	shares, err = validator.SharesFromTokens(amt)
	if err != nil {
		return shares, err
	}

	sharesTruncated, err := validator.SharesFromTokensTruncated(amt)
	if err != nil {
		return shares, err
	}

	delShares := del.GetShares()
	if sharesTruncated.GT(delShares) {
		return shares, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid shares amount")
	}

	tolerance, err := validator.SharesFromTokens(math.OneInt())
	if err != nil {
		return shares, err
	}

	if delShares.Sub(shares).LT(tolerance) {
		shares = delShares
	}

	return shares, nil
}

func (s msgServer) Undelegate(ctx context.Context, msg *stakingtypes.MsgUndelegate) (*stakingtypes.MsgUndelegateResponse, error) {
	addr, err := s.keeper.ValidatorAddressCodec().StringToBytes(msg.ValidatorAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid validator address: %s", err)
	}

	delegatorAddress, err := s.keeper.authKeeper.AddressCodec().StringToBytes(msg.DelegatorAddress)
	if err != nil {
		return nil, sdkerrors.ErrInvalidAddress.Wrapf("invalid delegator address: %s", err)
	}

	if !msg.Amount.IsValid() || !msg.Amount.Amount.IsPositive() {
		return nil, errorsmod.Wrap(
			sdkerrors.ErrInvalidRequest,
			"invalid shares amount",
		)
	}

	shares, err := s.keeper.ValidateUnbondAmount(
		ctx, delegatorAddress, addr, msg.Amount.Amount,
	)
	if err != nil {
		return nil, err
	}

	bondDenom, err := s.keeper.BondDenom(ctx)
	if err != nil {
		return nil, err
	}

	if msg.Amount.Denom != bondDenom {
		return nil, errorsmod.Wrapf(
			sdkerrors.ErrInvalidRequest, "invalid coin denomination: got %s, expected %s", msg.Amount.Denom, bondDenom,
		)
	}

	completionTime, undelegatedAmt, err := s.keeper.Undelegate(ctx, delegatorAddress, addr, shares)
	if err != nil {
		return nil, err
	}

	undelegatedCoin := sdk.NewCoin(msg.Amount.Denom, undelegatedAmt)

	if msg.Amount.Amount.IsInt64() {
		defer func() {
			telemetry.IncrCounter(1, stakingtypes.ModuleName, "undelegate") //nolint:staticcheck // TODO: switch to OpenTelemetry
			telemetry.SetGaugeWithLabels(                                   //nolint:staticcheck // TODO: switch to OpenTelemetry
				[]string{"tx", "msg", sdk.MsgTypeURL(msg)},
				float32(msg.Amount.Amount.Int64()),
				[]metrics.Label{telemetry.NewLabel("denom", msg.Amount.Denom)}, //nolint:staticcheck // TODO: switch to OpenTelemetry
			)
		}()
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			stakingtypes.EventTypeUnbond,
			sdk.NewAttribute(stakingtypes.AttributeKeyValidator, msg.ValidatorAddress),
			sdk.NewAttribute(stakingtypes.AttributeKeyDelegator, msg.DelegatorAddress),
			sdk.NewAttribute(sdk.AttributeKeyAmount, undelegatedCoin.String()),
			sdk.NewAttribute(stakingtypes.AttributeKeyCompletionTime, completionTime.Format(time.RFC3339)),
		),
	})

	return &stakingtypes.MsgUndelegateResponse{
		CompletionTime: completionTime,
		Amount:         undelegatedCoin,
	}, nil
}
