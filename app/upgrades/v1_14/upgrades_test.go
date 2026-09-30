package v1_14_test

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
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	xplaapp "github.com/xpladev/xpla/app"
	apphelpers "github.com/xpladev/xpla/app/helpers"
	v1_14 "github.com/xpladev/xpla/app/upgrades/v1_14"
	pauth "github.com/xpladev/xpla/precompile/auth"
	pbank "github.com/xpladev/xpla/precompile/bank"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
)

func TestApplyUpgradeSetsModuleParamsAndPreservesState(t *testing.T) {
	originalHome := xplaapp.DefaultNodeHome
	xplaapp.DefaultNodeHome = t.TempDir()
	t.Cleanup(func() { xplaapp.DefaultNodeHome = originalHome })

	app := apphelpers.Setup(t, "v1-14-upgrade")
	_, err := app.Commit()
	require.NoError(t, err)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.LastBlockHeight() + 1,
		Hash:   app.LastCommitID().Hash,
	})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	const upgradeHeight int64 = 100
	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{Height: upgradeHeight})

	distributionParams, err := app.DistrKeeper.Params.Get(ctx)
	require.NoError(t, err)
	distributionParams.CommunityTax = sdkmath.LegacyMustNewDecFromStr("0.37")
	distributionParams.WithdrawAddrEnabled = false
	require.NoError(t, distributionParams.ValidateBasic())
	require.NoError(t, app.DistrKeeper.Params.Set(ctx, distributionParams))

	rewardParams, err := app.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	rewardParams.FeePoolRate = sdkmath.LegacyMustNewDecFromStr("0.9")
	rewardParams.CommunityPoolRate = sdkmath.LegacyMustNewDecFromStr("0.1")
	rewardParams.ReserveRate = sdkmath.LegacyZeroDec()
	rewardParams.ReserveAccount = "xpla10ksn9528f82uwnmz3sgr4n42l0nucmzntjrg00"
	rewardParams.RewardDistributeAccount = "xpla19dacf8gzsvuj9txzw0wmtfpdg8swpd4jxl3ks2"
	require.NoError(t, app.RewardKeeper.SetParams(ctx, rewardParams))

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

	evmParams := app.EvmKeeper.GetParams(ctx)
	evmParams.ActiveStaticPrecompiles = []string{
		evmtypes.P256PrecompileAddress,
		evmtypes.Bech32PrecompileAddress,
		evmtypes.StakingPrecompileAddress,
		evmtypes.DistributionPrecompileAddress,
		evmtypes.GovPrecompileAddress,
		evmtypes.SlashingPrecompileAddress,
		pauth.Address.Hex(),
	}
	evmParams.EVMChannels = []string{"channel-7"}
	evmParams.HistoryServeWindow = 1234
	require.NoError(t, app.EvmKeeper.SetParams(ctx, evmParams))

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
	require.True(t, app.UpgradeKeeper.HasHandler(v1_14.UpgradeName))

	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, upgradetypes.Plan{
		Name:   v1_14.UpgradeName,
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

	updatedRewardParams, err := app.RewardKeeper.GetParams(ctx)
	require.NoError(t, err)
	expectedRewardParams := rewardParams
	expectedRewardParams.FeePoolRate = sdkmath.LegacyOneDec()
	expectedRewardParams.CommunityPoolRate = sdkmath.LegacyZeroDec()
	require.Equal(t, expectedRewardParams, updatedRewardParams)

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

	expectedEVMParams := evmParams
	expectedEVMParams.ActiveStaticPrecompiles = []string{
		evmtypes.P256PrecompileAddress,
		evmtypes.Bech32PrecompileAddress,
		evmtypes.StakingPrecompileAddress,
		evmtypes.DistributionPrecompileAddress,
		evmtypes.ICS20PrecompileAddress,
		evmtypes.GovPrecompileAddress,
		evmtypes.SlashingPrecompileAddress,
		pbank.Address.Hex(),
		pwasm.Address.Hex(),
		pauth.Address.Hex(),
	}
	updatedEVMParams := app.EvmKeeper.GetParams(ctx)
	require.Equal(t, expectedEVMParams, updatedEVMParams)
	require.NotContains(t, updatedEVMParams.ActiveStaticPrecompiles, pwasm.DelegatecallAddress.Hex())

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
}

func axpla(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(dynamicdeflationtypes.TargetDenom, amount)
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
