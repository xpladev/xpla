package v1_14

import (
	"context"
	"fmt"
	"slices"

	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/app/keepers"
	pauth "github.com/xpladev/xpla/precompile/auth"
	pbank "github.com/xpladev/xpla/precompile/bank"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

const targetGasPriceAxpla int64 = 10_000_000_000_000

// CreateUpgradeHandler creates the v1.14 binary upgrade handler.
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

		// Active precompile contract
		evmParams := appKeepers.EvmKeeper.GetParams(ctx)
		for _, address := range []string{
			pbank.Address.Hex(),
			pwasm.Address.Hex(),
			pwasm.DelegatecallAddress.Hex(),
			pauth.Address.Hex(),
			evmtypes.ICS20PrecompileAddress,
		} {
			if !slices.Contains(evmParams.ActiveStaticPrecompiles, address) {
				evmParams.ActiveStaticPrecompiles = append(evmParams.ActiveStaticPrecompiles, address)
			}
		}
		if err := appKeepers.EvmKeeper.SetParams(ctx, evmParams); err != nil {
			return nil, fmt.Errorf("set EVM params: %w", err)
		}

		// active dynamic deflation module
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

		rewardParams, err := appKeepers.RewardKeeper.GetParams(ctx)
		if err != nil {
			return nil, fmt.Errorf("get reward params: %w", err)
		}
		rewardParams.FeePoolRate = sdkmath.LegacyOneDec()
		rewardParams.CommunityPoolRate = sdkmath.LegacyZeroDec()
		rewardParams.ReserveRate = sdkmath.LegacyZeroDec()
		if err := appKeepers.RewardKeeper.SetParams(ctx, rewardParams); err != nil {
			return nil, fmt.Errorf("set reward params: %w", err)
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
