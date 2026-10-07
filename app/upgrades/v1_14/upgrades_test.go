package v1_14_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/collections"
	ccodec "cosmossdk.io/collections/codec"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256r1"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"

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
	ctx := app.NewUncachedContext(false, tmproto.Header{Height: upgradeHeight, Time: time.Unix(upgradeHeight, 0).UTC()})

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

	p256Key, err := secp256r1.GenPrivKey()
	require.NoError(t, err)
	longPubKey := p256Key.PubKey()
	longAddress := sdk.AccAddress(longPubKey.Address())
	require.Len(t, longAddress, 32)
	longAccount := app.AccountKeeper.NewAccountWithAddress(ctx, longAddress)
	require.NoError(t, longAccount.SetPubKey(longPubKey))
	require.NoError(t, longAccount.SetSequence(1))
	app.AccountKeeper.SetAccount(ctx, longAccount)
	longAccountNumber := longAccount.GetAccountNumber()
	longSuffix := sdk.AccAddress(bytes.Clone(longAddress[len(longAddress)-20:]))

	creator := sdk.AccAddress(bytes.Repeat([]byte{0x71}, 20))
	creatorAccount := app.AccountKeeper.NewAccountWithAddress(ctx, creator)
	require.NoError(t, creatorAccount.SetSequence(7))
	app.AccountKeeper.SetAccount(ctx, creatorAccount)
	longBalance := sdk.NewCoins(axpla(13))
	creatorBalance := sdk.NewCoins(axpla(17))

	require.NoError(t, app.BankKeeper.MintCoins(ctx, minttypes.ModuleName, sdk.NewCoins(
		axpla(137),
		sdk.NewInt64Coin("ufoo", 3),
	)))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, longAddress, longBalance))
	require.NoError(t, app.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, creator, creatorBalance))
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

	wasmCode, err := os.ReadFile(filepath.Join("..", "..", "..", "tests", "solidity", "suites", "misc", "any_dispatch.wasm"))
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&app.WasmKeeper)
	codeID, _, err := contractKeeper.Create(ctx, creator, wasmCode, nil)
	require.NoError(t, err)
	contractAddress, _, err := contractKeeper.Instantiate(ctx, codeID, creator, nil, []byte(`{}`), "v1.14 alias rebuild", nil)
	require.NoError(t, err)
	require.Len(t, contractAddress, 32)
	contractAlias := sdk.AccAddress(bytes.Clone(contractAddress[len(contractAddress)-20:]))
	app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, contractAlias))

	govAlias := app.AccountKeeper.GetModuleAddress("gov")
	maliciousGovTarget := sdk.AccAddress(append(bytes.Repeat([]byte{0x72}, 12), govAlias...))
	arbitraryAlias := sdk.AccAddress(bytes.Repeat([]byte{0x73}, 20))
	arbitraryTarget := sdk.AccAddress(append(bytes.Repeat([]byte{0x74}, 12), arbitraryAlias...))
	legacyAliases := collections.NewMap(
		collections.NewSchemaBuilder(runtime.NewKVStoreService(app.GetKey(authtypes.StoreKey))),
		collections.NewPrefix("sliceAddress"), "legacy_slice_address",
		sdk.AccAddressKey, ccodec.KeyToValueCodec(sdk.AccAddressKey),
	)
	require.NoError(t, legacyAliases.Set(ctx, govAlias, maliciousGovTarget))
	require.NoError(t, legacyAliases.Set(ctx, arbitraryAlias, arbitraryTarget))
	require.NoError(t, legacyAliases.Set(ctx, contractAlias, arbitraryTarget))
	require.NoError(t, legacyAliases.Set(ctx, longSuffix, longAddress))
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, longSuffix))
	require.Nil(t, app.WasmKeeper.GetContractInfo(ctx, longAddress))
	require.Equal(t, longBalance, app.BankKeeper.GetAllBalances(ctx, longAddress))
	require.Equal(t, uint64(7), app.AccountKeeper.GetAccount(ctx, creator).GetSequence())
	require.Equal(t, creatorBalance, app.BankKeeper.GetAllBalances(ctx, creator))

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

	resolvedContract, found, err := app.WasmKeeper.ResolveWasmAlias(ctx, contractAlias)
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, contractAddress, resolvedContract)
	require.Equal(t, contractAlias, app.AccountKeeper.GetAccount(ctx, contractAlias).GetAddress(), "exact EOA remains authoritative")
	_, found, err = app.WasmKeeper.ResolveWasmAlias(ctx, arbitraryAlias)
	require.NoError(t, err)
	require.False(t, found)
	require.Equal(t, 1, countAliasEntries(t, ctx.KVStore(app.GetKey(wasmtypes.StoreKey)), "wasmAlias"), "only the authoritative contract mapping survives")
	_, found, err = app.WasmKeeper.ResolveWasmAlias(ctx, longSuffix)
	require.NoError(t, err)
	require.False(t, found, "non-Wasm SDK account must not gain a Wasm alias")
	require.Zero(t, countAliasEntries(t, ctx.KVStore(app.GetKey(authtypes.StoreKey)), "sliceAddress"), "all legacy auth mappings must be removed")
	preservedLongAccount := app.AccountKeeper.GetAccount(ctx, longAddress)
	require.NotNil(t, preservedLongAccount)
	require.Equal(t, longAddress, preservedLongAccount.GetAddress())
	require.True(t, longPubKey.Equals(preservedLongAccount.GetPubKey()))
	require.Equal(t, longAccountNumber, preservedLongAccount.GetAccountNumber())
	require.Equal(t, uint64(1), preservedLongAccount.GetSequence())
	require.Equal(t, longBalance, app.BankKeeper.GetAllBalances(ctx, longAddress))
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, longSuffix), "upgrade must not create a suffix account")
	require.Zero(t, app.EvmKeeper.GetNonce(ctx, common.BytesToAddress(longSuffix)))
	require.Equal(t, uint64(7), app.AccountKeeper.GetAccount(ctx, creator).GetSequence())
	require.Equal(t, creatorBalance, app.BankKeeper.GetAllBalances(ctx, creator))
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

func countAliasEntries(t *testing.T, store storetypes.KVStore, prefix string) int {
	t.Helper()
	iterator := storetypes.KVStorePrefixIterator(store, collections.NewPrefix(prefix).Bytes())
	defer func() { require.NoError(t, iterator.Close()) }()
	count := 0
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	require.NoError(t, iterator.Error())
	return count
}
