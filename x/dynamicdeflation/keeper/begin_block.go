package keeper

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

// BeginBlock atomically routes the FeeCollector balance and, at the period boundary, settles it.
func (k Keeper) BeginBlock(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if sdkCtx.BlockHeight() == 1 {
		return nil
	}

	cacheCtx, write := sdkCtx.CacheContext()
	if err := k.routeAndMaybeSettle(cacheCtx); err != nil {
		return err
	}
	write()
	return nil
}

func (k Keeper) routeAndMaybeSettle(ctx sdk.Context) error {
	period, err := k.getOrCreatePeriod(ctx)
	if err != nil {
		return err
	}
	if period == nil {
		return nil
	}
	if ctx.BlockHeight() > period.EndHeight {
		return fmt.Errorf("current height %d is after period end height %d", ctx.BlockHeight(), period.EndHeight)
	}

	feeCollector := k.accountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	if feeCollector == nil {
		return fmt.Errorf("%s module account has not been set", authtypes.FeeCollectorName)
	}
	gross := k.bankKeeper.GetBalance(ctx, feeCollector, types.TargetDenom).Amount
	allocated, err := CalculateAllocatedAmount(gross, period.ActiveConfig.AllocationRate)
	if err != nil {
		return err
	}
	if allocated.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(types.TargetDenom, allocated))
		if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, types.PoolName, coins); err != nil {
			return fmt.Errorf("route allocated amount: %w", err)
		}
	}

	period.GrossAmount, err = period.GrossAmount.SafeAdd(gross)
	if err != nil {
		return fmt.Errorf("accumulate period gross amount: %w", err)
	}
	period.AllocatedAmount, err = period.AllocatedAmount.SafeAdd(allocated)
	if err != nil {
		return fmt.Errorf("accumulate period allocated amount: %w", err)
	}
	if err := k.CurrentPeriodStore.Set(ctx, *period); err != nil {
		return err
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeRouted,
		sdk.NewAttribute(types.AttributeKeyHeight, strconv.FormatInt(ctx.BlockHeight(), 10)),
		sdk.NewAttribute(types.AttributeKeyGross, gross.String()),
		sdk.NewAttribute(types.AttributeKeyAllocated, allocated.String()),
		sdk.NewAttribute(types.AttributeKeyRemainder, gross.Sub(allocated).String()),
	))

	if ctx.BlockHeight() < period.EndHeight {
		return nil
	}
	return k.settle(ctx, *period)
}

func (k Keeper) getOrCreatePeriod(ctx context.Context) (*types.CurrentPeriod, error) {
	period, err := k.CurrentPeriodStore.Get(ctx)
	if err == nil {
		return &period, nil
	}
	if !errors.Is(err, collections.ErrNotFound) {
		return nil, err
	}
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}
	if !params.Enabled {
		return nil, nil
	}
	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	endHeight, err := types.CalculateSettlementEndHeight(height, params.SettlementIntervalBlocks)
	if err != nil {
		return nil, err
	}
	period = types.CurrentPeriod{
		StartHeight:     height,
		EndHeight:       endHeight,
		ActiveConfig:    params,
		GrossAmount:     sdkmath.ZeroInt(),
		AllocatedAmount: sdkmath.ZeroInt(),
	}
	if err := k.CurrentPeriodStore.Set(ctx, period); err != nil {
		return nil, err
	}
	return &period, nil
}
