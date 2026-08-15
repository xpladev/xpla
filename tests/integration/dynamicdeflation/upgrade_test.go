package dynamicdeflation_test

import (
	"bytes"
	"testing"

	"cosmossdk.io/collections"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	xplaapp "github.com/xpladev/xpla/app"
	v1_12 "github.com/xpladev/xpla/app/upgrades/v1_12"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func verifyV112UpgradeLifecycle(t *testing.T, app *xplaapp.XplaApp) {
	t.Helper()
	const upgradeHeight int64 = 100
	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{Height: upgradeHeight})

	distributionParams, err := app.DistrKeeper.Params.Get(ctx)
	require.NoError(t, err)
	distributionParams.CommunityTax = sdkmath.LegacyMustNewDecFromStr("0.37")
	distributionParams.WithdrawAddrEnabled = false
	require.NoError(t, distributionParams.ValidateBasic())
	require.NoError(t, app.DistrKeeper.Params.Set(ctx, distributionParams))

	feeMarketParams := app.FeeMarketKeeper.GetParams(ctx)
	feeMarketParams.NoBaseFee = true
	feeMarketParams.BaseFeeChangeDenominator = 13
	feeMarketParams.ElasticityMultiplier = 3
	feeMarketParams.BaseFee = sdkmath.LegacyNewDec(77)
	feeMarketParams.EnableHeight = 7
	feeMarketParams.MinGasPrice = sdkmath.LegacyNewDec(11)
	feeMarketParams.MinGasMultiplier = sdkmath.LegacyMustNewDecFromStr("0.75")
	require.NoError(t, feeMarketParams.Validate())
	require.NoError(t, app.FeeMarketKeeper.SetParams(ctx, feeMarketParams))

	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, sdk.NewCoins(
		axpla(107),
		sdk.NewInt64Coin("ufoo", 3),
	)))
	mintAddress := app.AccountKeeper.GetModuleAddress(minttypes.ModuleName)
	require.NoError(t, app.DistrKeeper.FundCommunityPool(ctx, sdk.NewCoins(axpla(7)), mintAddress))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToModule(
		ctx,
		minttypes.ModuleName,
		authtypes.FeeCollectorName,
		sdk.NewCoins(axpla(100), sdk.NewInt64Coin("ufoo", 3)),
	))

	feeCollectorAddress := app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName)
	feeCollectorBefore := app.BankKeeper.GetAllBalances(ctx, feeCollectorAddress)
	feePoolBefore, err := app.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	axplaSupplyBefore := app.BankKeeper.GetSupply(ctx, dynamicdeflationtypes.TargetDenom)
	otherSupplyBefore := app.BankKeeper.GetSupply(ctx, "ufoo")
	distributionStateBefore := snapshotStoreExcluding(
		t,
		ctx.KVStore(app.GetKey(distrtypes.StoreKey)),
		distrtypes.ParamsKey.Bytes(),
	)
	require.True(t, hasValidatorRewardState(distributionStateBefore))

	clearStore(t, ctx.KVStore(app.GetKey(dynamicdeflationtypes.StoreKey)))
	_, err = app.DynamicDeflationKeeper.GetParams(ctx)
	require.ErrorIs(t, err, collections.ErrNotFound)

	versionKey := append([]byte{upgradetypes.VersionMapByte}, []byte(dynamicdeflationtypes.ModuleName)...)
	ctx.KVStore(app.GetKey(upgradetypes.StoreKey)).Delete(versionKey)
	versionMap, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	_, exists := versionMap[dynamicdeflationtypes.ModuleName]
	require.False(t, exists)
	require.True(t, app.UpgradeKeeper.HasHandler(v1_12.UpgradeName))

	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, upgradetypes.Plan{
		Name:   v1_12.UpgradeName,
		Height: upgradeHeight,
	}))

	dynamicParams, err := app.DynamicDeflationKeeper.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, dynamicdeflationtypes.DefaultParams(), dynamicParams)
	require.Equal(t, uint64(100_000), dynamicParams.SettlementIntervalBlocks)
	require.True(t, dynamicParams.AllocationRate.Equal(sdkmath.LegacyMustNewDecFromStr("0.20")))
	require.Equal(t, "69444000000000000000000", dynamicParams.MinFeeAmount.Amount.String())
	require.Equal(t, "3472222000000000000000000", dynamicParams.MaxFeeAmount.Amount.String())
	hasCurrentPeriod, err := app.DynamicDeflationKeeper.CurrentPeriodStore.Has(ctx)
	require.NoError(t, err)
	require.False(t, hasCurrentPeriod)

	updatedVersionMap, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), updatedVersionMap[dynamicdeflationtypes.ModuleName])

	updatedDistributionParams, err := app.DistrKeeper.Params.Get(ctx)
	require.NoError(t, err)
	expectedDistributionParams := distributionParams
	expectedDistributionParams.CommunityTax = sdkmath.LegacyZeroDec()
	require.Equal(t, expectedDistributionParams, updatedDistributionParams)

	updatedFeeMarketParams := app.FeeMarketKeeper.GetParams(ctx)
	expectedGasPrice := sdkmath.LegacyNewDec(10_000_000_000_000)
	expectedFeeMarketParams := feeMarketParams
	expectedFeeMarketParams.NoBaseFee = false
	expectedFeeMarketParams.EnableHeight = upgradeHeight
	expectedFeeMarketParams.MinGasPrice = expectedGasPrice
	expectedFeeMarketParams.BaseFee = expectedGasPrice
	require.Equal(t, expectedFeeMarketParams, updatedFeeMarketParams)
	require.Equal(t,
		sdkmath.NewIntWithDecimal(1, 18),
		updatedFeeMarketParams.MinGasPrice.
			MulInt(sdkmath.NewIntFromUint64(100_000)).
			TruncateInt(),
	)
	require.True(t, app.FeeMarketKeeper.CalculateBaseFee(ctx).Equal(expectedGasPrice))

	require.Equal(t, feeCollectorBefore, app.BankKeeper.GetAllBalances(ctx, feeCollectorAddress))
	feePoolAfter, err := app.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, feePoolBefore, feePoolAfter)
	require.Equal(t, axplaSupplyBefore, app.BankKeeper.GetSupply(ctx, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, otherSupplyBefore, app.BankKeeper.GetSupply(ctx, "ufoo"))
	require.Equal(t, distributionStateBefore, snapshotStoreExcluding(
		t,
		ctx.KVStore(app.GetKey(distrtypes.StoreKey)),
		distrtypes.ParamsKey.Bytes(),
	))

	validators, err := app.StakingKeeper.GetAllValidators(ctx)
	require.NoError(t, err)
	require.Len(t, validators, 1)
	consAddress, err := validators[0].GetConsAddr()
	require.NoError(t, err)
	ctx = ctx.WithVoteInfos([]abci.VoteInfo{{
		Validator: abci.Validator{
			Address: consAddress,
			Power:   validators[0].GetConsensusPower(sdk.DefaultPowerReduction),
		},
		BlockIdFlag: tmproto.BlockIDFlagCommit,
	}})
	app.DistrKeeper.SetPreviousProposerConsAddr(ctx, consAddress)
	feePoolBeforeDistribution, err := app.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	supplyBeforeSettlement := app.BankKeeper.GetSupply(ctx, dynamicdeflationtypes.TargetDenom).Amount

	require.NoError(t, app.DynamicDeflationKeeper.BeginBlock(ctx))
	period, err := app.DynamicDeflationKeeper.CurrentPeriodStore.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, upgradeHeight, period.StartHeight)
	require.Equal(t, int64(100_000), period.EndHeight)
	require.Equal(t, sdkmath.NewInt(100), period.GrossAmount)
	require.Equal(t, sdkmath.NewInt(20), period.AllocatedAmount)

	require.NoError(t, distribution.BeginBlocker(ctx, app.DistrKeeper))
	feePoolAfterDistribution, err := app.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, feePoolBeforeDistribution.CommunityPool, feePoolAfterDistribution.CommunityPool)

	ctx = ctx.WithBlockHeight(period.EndHeight - 1).WithVoteInfos(nil)
	require.NoError(t, app.DynamicDeflationKeeper.BeginBlock(ctx))
	require.Empty(t, eventAttributesByType(ctx, dynamicdeflationtypes.EventTypeSettled))

	ctx = ctx.WithBlockHeight(period.EndHeight)
	require.NoError(t, app.DynamicDeflationKeeper.BeginBlock(ctx))
	events := eventAttributesByType(ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "100", events[0][dynamicdeflationtypes.AttributeKeyStartHeight])
	require.Equal(t, "100000", events[0][dynamicdeflationtypes.AttributeKeySettlementHeight])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyBurn])
	require.Equal(t, "0", events[0][dynamicdeflationtypes.AttributeKeyCommunity])
	require.Equal(t, sdkmath.NewInt(20), supplyBeforeSettlement.Sub(
		app.BankKeeper.GetSupply(ctx, dynamicdeflationtypes.TargetDenom).Amount,
	))
}

func clearStore(t *testing.T, store storetypes.KVStore) {
	t.Helper()
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	var keys [][]byte
	for ; iterator.Valid(); iterator.Next() {
		keys = append(keys, bytes.Clone(iterator.Key()))
	}
	for _, key := range keys {
		store.Delete(key)
	}
}

func snapshotStoreExcluding(t *testing.T, store storetypes.KVStore, excludedKey []byte) map[string][]byte {
	t.Helper()
	iterator := store.Iterator(nil, nil)
	defer iterator.Close()
	snapshot := make(map[string][]byte)
	for ; iterator.Valid(); iterator.Next() {
		if bytes.Equal(iterator.Key(), excludedKey) {
			continue
		}
		snapshot[string(iterator.Key())] = bytes.Clone(iterator.Value())
	}
	return snapshot
}

func hasValidatorRewardState(snapshot map[string][]byte) bool {
	for key := range snapshot {
		if len(key) == 0 {
			continue
		}
		switch key[0] {
		case 0x02, 0x05, 0x06, 0x07:
			return true
		}
	}
	return false
}
