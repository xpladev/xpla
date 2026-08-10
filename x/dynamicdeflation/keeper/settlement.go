package keeper

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func (k Keeper) settle(ctx context.Context, period types.CurrentPeriod) error {
	poolBalance := k.bankKeeper.GetBalance(ctx, k.poolAddress, types.TargetDenom).Amount
	if poolBalance.LT(period.AllocatedAmount) {
		return fmt.Errorf(
			"dynamic deflation pool account invariant violated: balance %s is less than allocated amount %s",
			poolBalance, period.AllocatedAmount,
		)
	}

	burn, community, err := CalculateSettlement(
		period.GrossAmount,
		period.AllocatedAmount,
		period.ActiveConfig.MinFeeAmount.Amount,
		period.ActiveConfig.MaxFeeAmount.Amount,
	)
	if err != nil {
		return err
	}
	if community.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(types.TargetDenom, community))
		if err := k.distKeeper.FundCommunityPool(ctx, coins, k.poolAddress); err != nil {
			return fmt.Errorf("fund community pool: %w", err)
		}
	}
	if burn.IsPositive() {
		coins := sdk.NewCoins(sdk.NewCoin(types.TargetDenom, burn))
		if err := k.bankKeeper.BurnCoins(ctx, types.PoolName, coins); err != nil {
			return fmt.Errorf("burn dynamic deflation amount: %w", err)
		}
	}

	height := sdk.UnwrapSDKContext(ctx).BlockHeight()
	if err := k.CurrentPeriodStore.Remove(ctx); err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
		types.EventTypeSettled,
		sdk.NewAttribute(types.AttributeKeyStartHeight, strconv.FormatInt(period.StartHeight, 10)),
		sdk.NewAttribute(types.AttributeKeyEndHeight, strconv.FormatInt(period.EndHeight, 10)),
		sdk.NewAttribute(types.AttributeKeySettlementHeight, strconv.FormatInt(height, 10)),
		sdk.NewAttribute(types.AttributeKeyGross, period.GrossAmount.String()),
		sdk.NewAttribute(types.AttributeKeyAllocated, period.AllocatedAmount.String()),
		sdk.NewAttribute(types.AttributeKeyBurn, burn.String()),
		sdk.NewAttribute(types.AttributeKeyCommunity, community.String()),
	))
	return nil
}
