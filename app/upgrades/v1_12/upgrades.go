package v1_12

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/xpladev/xpla/app/keepers"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

const targetGasPriceAxpla int64 = 10_000_000_000_000

// CreateUpgradeHandler creates the v1.12 binary upgrade handler.
func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	appKeepers *keepers.AppKeepers,
	_ codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		updatedVM, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return nil, err
		}

		params, err := appKeepers.DistrKeeper.Params.Get(ctx)
		if err != nil {
			return nil, fmt.Errorf("get distribution params: %w", err)
		}
		params.CommunityTax = sdkmath.LegacyZeroDec()
		if err := params.ValidateBasic(); err != nil {
			return nil, fmt.Errorf("validate distribution params: %w", err)
		}
		if err := appKeepers.DistrKeeper.Params.Set(ctx, params); err != nil {
			return nil, fmt.Errorf("set distribution params: %w", err)
		}

		// A transfer with a 100,000 gas limit must pay 1 XPLA (10^18 axpla):
		// 10^18 axpla / 100,000 gas = 10^13 axpla/gas.
		targetGasPrice := sdkmath.LegacyNewDec(targetGasPriceAxpla)
		feeMarketParams := appKeepers.FeeMarketKeeper.GetParams(ctx)
		feeMarketParams.NoBaseFee = false
		feeMarketParams.EnableHeight = ctx.BlockHeight()
		feeMarketParams.MinGasPrice = targetGasPrice
		feeMarketParams.BaseFee = targetGasPrice
		if err := feeMarketParams.Validate(); err != nil {
			return nil, fmt.Errorf("validate fee market params: %w", err)
		}
		if err := appKeepers.FeeMarketKeeper.SetParams(ctx, feeMarketParams); err != nil {
			return nil, fmt.Errorf("set fee market params: %w", err)
		}

		dynamicDeflationParams := dynamicdeflationtypes.DefaultParams()
		if err := dynamicDeflationParams.Validate(); err != nil {
			return nil, fmt.Errorf("validate dynamic deflation params: %w", err)
		}
		if err := appKeepers.DynamicDeflationKeeper.SetParams(ctx, dynamicDeflationParams); err != nil {
			return nil, fmt.Errorf("set dynamic deflation params: %w", err)
		}

		return updatedVM, nil
	}
}
