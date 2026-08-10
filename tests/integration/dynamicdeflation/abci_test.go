package dynamicdeflation_test

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"

	xplaapp "github.com/xpladev/xpla/app"
	apphelpers "github.com/xpladev/xpla/app/helpers"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
	"github.com/xpladev/xpla/x/reward"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
)

func TestFreshGenesisAndV112UpgradeAppLifecycles(t *testing.T) {
	originalHome := xplaapp.DefaultNodeHome
	xplaapp.DefaultNodeHome = t.TempDir()
	t.Cleanup(func() { xplaapp.DefaultNodeHome = originalHome })

	app := apphelpers.Setup(t, "dynamic-deflation-integration")
	_, err := app.Commit()
	require.NoError(t, err)
	require.Equal(t, int64(1), app.LastBlockHeight())

	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: app.LastBlockHeight() + 1,
		Hash:   app.LastCommitID().Hash,
	})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)

	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{Height: 2, Time: time.Unix(2, 0).UTC()})
	period, err := app.DynamicDeflationKeeper.CurrentPeriodStore.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), period.StartHeight)
	require.Equal(t, int64(100_000), period.EndHeight)
	require.True(t, period.GrossAmount.IsPositive(), "height 2 mint inflow was not observed")
	require.True(t, period.AllocatedAmount.IsPositive(), "height 2 mint inflow was not routed")
	require.True(t, app.BankKeeper.GetBalance(
		ctx,
		app.AccountKeeper.GetModuleAddress(authtypes.FeeCollectorName),
		dynamicdeflationtypes.TargetDenom,
	).Amount.IsZero(), "distribution did not consume the FeeCollector remainder")

	verifyV112UpgradeLifecycle(t, app)
}

func TestBeginBlockAtHeightOneDoesNotCreatePeriodOrRouteFees(t *testing.T) {
	fixture := newFixture(t)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(1))
	require.NoError(t, err)

	require.Equal(t, sdkmath.NewInt(100), fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.ZeroInt(), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))
	hasPeriod, err := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Has(fixture.ctx)
	require.NoError(t, err)
	require.False(t, hasPeriod)
}

func TestBeginBlockRoutesOnlyTwentyPercentOfAxplaBeforeDistribution(t *testing.T) {
	fixture := newFixture(t)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100), sdk.NewInt64Coin("ufoo", 20)))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2))
	require.NoError(t, err)

	require.Equal(t, sdkmath.NewInt(80), fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.NewInt(20), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))
	require.Nil(t, fixture.app.AccountKeeper.GetModuleAddress(dynamicdeflationtypes.ModuleName))
	require.Equal(t, sdkmath.NewInt(20), fixture.moduleBalance(authtypes.FeeCollectorName, "ufoo"))
	require.Equal(t, sdkmath.ZeroInt(), fixture.moduleBalance(dynamicdeflationtypes.PoolName, "ufoo"))

	period, err := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Get(fixture.ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), period.StartHeight)
	require.Equal(t, int64(100_000), period.EndHeight)
	require.Equal(t, sdkmath.NewInt(100), period.GrossAmount)
	require.Equal(t, sdkmath.NewInt(20), period.AllocatedAmount)
}

func TestZeroPowerDistributionConsumesRemainderAndNonAxplaAfterDynamicRouting(t *testing.T) {
	fixture := newFixture(t)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100), sdk.NewInt64Coin("ufoo", 20)))
	ctx := fixture.atHeight(2).WithVoteInfos(nil)
	fixture.ctx = ctx

	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(ctx))
	require.Equal(t, sdkmath.NewInt(20), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.NewInt(80), fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.NewInt(20), fixture.moduleBalance(authtypes.FeeCollectorName, "ufoo"))

	require.NoError(t, distribution.BeginBlocker(ctx, fixture.app.DistrKeeper))
	require.True(t, fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom).IsZero())
	require.True(t, fixture.moduleBalance(authtypes.FeeCollectorName, "ufoo").IsZero())
	require.Equal(t, sdkmath.NewInt(20), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))

	feePool, err := fixture.app.DistrKeeper.FeePool.Get(ctx)
	require.NoError(t, err)
	communityPool, _ := feePool.CommunityPool.TruncateDecimal()
	require.Equal(t, sdkmath.NewInt(80), communityPool.AmountOf(dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.NewInt(20), communityPool.AmountOf("ufoo"))
}

func TestDefaultIntervalSettlesAtGlobalHundredThousandBlockBoundary(t *testing.T) {
	fixture := newFixture(t)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))

	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2)))
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(99_999)))
	require.Empty(t, eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled))
	period, err := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Get(fixture.ctx)
	require.NoError(t, err)
	require.Equal(t, int64(2), period.StartHeight)
	require.Equal(t, int64(100_000), period.EndHeight)

	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(5)))
	finalBlockGross := fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom)
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(100_000)))
	events := eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "2", events[0][dynamicdeflationtypes.AttributeKeyStartHeight])
	require.Equal(t, "100000", events[0][dynamicdeflationtypes.AttributeKeyEndHeight])
	require.Equal(t, "100000", events[0][dynamicdeflationtypes.AttributeKeySettlementHeight])
	require.Equal(t, period.GrossAmount.Add(finalBlockGross).String(), events[0][dynamicdeflationtypes.AttributeKeyGross])
}

func TestIntervalOneSettlesAfterRoutingAndBurnsAllocatedAmount(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(1, 1_000, 10_000))
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))
	supplyBefore := fixture.app.BankKeeper.GetSupply(fixture.ctx, dynamicdeflationtypes.TargetDenom).Amount

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2))
	require.NoError(t, err)

	supplyAfter := fixture.app.BankKeeper.GetSupply(fixture.ctx, dynamicdeflationtypes.TargetDenom).Amount
	require.Equal(t, sdkmath.NewInt(20), supplyBefore.Sub(supplyAfter))
	require.Equal(t, sdkmath.ZeroInt(), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))
	require.Equal(t, sdkmath.NewInt(80), fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))

	events := eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "2", events[0][dynamicdeflationtypes.AttributeKeyStartHeight])
	require.Equal(t, "2", events[0][dynamicdeflationtypes.AttributeKeyEndHeight])
	require.Equal(t, "100", events[0][dynamicdeflationtypes.AttributeKeyGross])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyAllocated])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyBurn])
	require.Equal(t, "0", events[0][dynamicdeflationtypes.AttributeKeyCommunity])
}

func TestSettlementEventContainsAuditAmounts(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(1, 1_000, 10_000))
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2))
	require.NoError(t, err)

	events := eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.NotContains(t, events[0], "period_id")
	require.Equal(t, "2", events[0][dynamicdeflationtypes.AttributeKeySettlementHeight])
	require.Equal(t, "100", events[0][dynamicdeflationtypes.AttributeKeyGross])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyAllocated])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyBurn])
	require.Equal(t, "0", events[0][dynamicdeflationtypes.AttributeKeyCommunity])
}

func TestCommunitySettlementAddsToExistingCommunityPoolBalance(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(1, 0, 1))
	fixture.fundCommunityPool(t, axpla(7))
	communityBefore := fixture.communityPoolAmount(t)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2))
	require.NoError(t, err)

	communityAfter := fixture.communityPoolAmount(t)
	require.Equal(t, sdkmath.NewInt(20), communityAfter.Sub(communityBefore))
	require.Equal(t, sdkmath.NewInt(27), communityAfter)
	events := eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "0", events[0][dynamicdeflationtypes.AttributeKeyBurn])
	require.Equal(t, "20", events[0][dynamicdeflationtypes.AttributeKeyCommunity])
}

func TestCommunityPoolQueryDirectFundAndGovernanceSpendRemainCompatible(t *testing.T) {
	fixture := newFixture(t)
	fixture.fundCommunityPool(t, axpla(30))

	querier := distrkeeper.NewQuerier(fixture.app.DistrKeeper)
	queryResponse, err := querier.CommunityPool(fixture.ctx, &distrtypes.QueryCommunityPoolRequest{})
	require.NoError(t, err)
	queryPool, _ := queryResponse.Pool.TruncateDecimal()
	require.Equal(t, sdkmath.NewInt(30), queryPool.AmountOf(dynamicdeflationtypes.TargetDenom))

	recipient := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	msgServer := distrkeeper.NewMsgServerImpl(fixture.app.DistrKeeper)
	_, err = msgServer.CommunityPoolSpend(fixture.ctx, &distrtypes.MsgCommunityPoolSpend{
		Authority: fixture.app.AccountKeeper.GetModuleAddress(govtypes.ModuleName).String(),
		Recipient: recipient.String(),
		Amount:    sdk.NewCoins(axpla(11)),
	})
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(11), fixture.app.BankKeeper.GetBalance(
		fixture.ctx,
		recipient,
		dynamicdeflationtypes.TargetDenom,
	).Amount)

	queryResponse, err = querier.CommunityPool(fixture.ctx, &distrtypes.QueryCommunityPoolRequest{})
	require.NoError(t, err)
	queryPool, _ = queryResponse.Pool.TruncateDecimal()
	require.Equal(t, sdkmath.NewInt(19), queryPool.AmountOf(dynamicdeflationtypes.TargetDenom))
}

func TestRewardFeeCollectorInflowIsRoutedOnceOnTheNextBlock(t *testing.T) {
	fixture := newFixture(t)
	rewardParams := rewardtypes.DefaultParams()
	rewardParams.RewardDistributeAccount = sdk.AccAddress(bytes.Repeat([]byte{2}, 20)).String()
	require.NoError(t, fixture.app.RewardKeeper.SetParams(fixture.ctx, rewardParams))

	blocksPerYear, err := fixture.app.RewardKeeper.GetBlocksPerYear(fixture.ctx)
	require.NoError(t, err)
	require.NotZero(t, blocksPerYear)
	totalRewardPool := sdkmath.NewIntFromUint64(blocksPerYear).MulRaw(100)
	fixture.fundModule(t, rewardtypes.ModuleName, sdk.NewCoins(sdk.NewCoin(dynamicdeflationtypes.TargetDenom, totalRewardPool)))

	ctx := fixture.atHeight(2).WithVoteInfos(nil)
	fixture.ctx = ctx
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(ctx))
	require.NoError(t, distribution.BeginBlocker(ctx, fixture.app.DistrKeeper))
	require.NoError(t, reward.BeginBlocker(ctx, fixture.app.RewardKeeper, fixture.app.BankKeeper, fixture.app.StakingKeeper, fixture.app.DistrKeeper))
	require.Equal(t, sdkmath.NewInt(100), fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))

	ctx = fixture.atHeight(3).WithVoteInfos(nil)
	fixture.ctx = ctx
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(ctx))
	period, err := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(100), period.GrossAmount)
	require.Equal(t, sdkmath.NewInt(20), period.AllocatedAmount)
	require.NoError(t, distribution.BeginBlocker(ctx, fixture.app.DistrKeeper))
	require.NoError(t, reward.BeginBlocker(ctx, fixture.app.RewardKeeper, fixture.app.BankKeeper, fixture.app.StakingKeeper, fixture.app.DistrKeeper))

	secondInflow := totalRewardPool.SubRaw(100).Quo(sdkmath.NewIntFromUint64(blocksPerYear))
	require.Equal(t, secondInflow, fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))
	ctx = fixture.atHeight(4).WithVoteInfos(nil)
	fixture.ctx = ctx
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(ctx))
	period, err = fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Get(ctx)
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(100).Add(secondInflow), period.GrossAmount)
	require.Equal(t, sdkmath.NewInt(20).Add(secondInflow.QuoRaw(5)), period.AllocatedAmount)
}

func TestSettlementExcludesSurplusFromBurn(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(1, 1_000, 10_000))
	fixture.fundModule(t, dynamicdeflationtypes.PoolName, sdk.NewCoins(axpla(7)))
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(2))
	require.NoError(t, err)

	require.Equal(t, sdkmath.NewInt(7), fixture.moduleBalance(dynamicdeflationtypes.PoolName, dynamicdeflationtypes.TargetDenom))
	status, err := fixture.app.DynamicDeflationKeeper.Status(fixture.ctx, &dynamicdeflationtypes.QueryStatusRequest{})
	require.NoError(t, err)
	require.Equal(t, sdkmath.NewInt(7), status.ModuleBalance.Amount)
	require.True(t, status.AllocatedAmount.Amount.IsZero())
	require.Equal(t, sdkmath.NewInt(7), status.SurplusAmount.Amount)
}

func TestSettlementFailsClosedWhenAllocatedBalanceIsDeficient(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(2, 1_000, 10_000))
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(3)))
	require.Empty(t, eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled))
	require.NoError(t, fixture.app.BankKeeper.SendCoinsFromModuleToModule(
		fixture.ctx,
		authtypes.FeeCollectorName,
		minttypes.ModuleName,
		sdk.NewCoins(axpla(80)),
	))
	require.NoError(t, fixture.app.BankKeeper.BurnCoins(fixture.ctx, dynamicdeflationtypes.PoolName, sdk.NewCoins(axpla(1))))

	err := fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(4))
	require.ErrorContains(t, err, "pool account invariant violated")

	period, getErr := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Get(fixture.ctx)
	require.NoError(t, getErr)
	require.Equal(t, sdkmath.NewInt(100), period.GrossAmount)
	require.Equal(t, sdkmath.NewInt(20), period.AllocatedAmount)
	require.Empty(t, eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled))
	status, queryErr := fixture.app.DynamicDeflationKeeper.Status(fixture.ctx, &dynamicdeflationtypes.QueryStatusRequest{})
	require.NoError(t, queryErr)
	require.Equal(t, sdkmath.NewInt(1), status.DeficitAmount.Amount)
}

func TestParamsChangedMidPeriodApplyOnlyAfterSettlement(t *testing.T) {
	fixture := newFixture(t)
	fixture.setParams(t, paramsForSettlement(2, 1_000, 10_000))
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(3)))
	require.Empty(t, eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled))
	require.NoError(t, fixture.app.BankKeeper.SendCoinsFromModuleToModule(
		fixture.ctx,
		authtypes.FeeCollectorName,
		minttypes.ModuleName,
		sdk.NewCoins(axpla(80)),
	))

	nextParams := paramsForSettlement(1, 0, 1)
	nextParams.Enabled = false
	nextParams.AllocationRate = sdkmath.LegacyOneDec()
	fixture.setParams(t, nextParams)
	fixture.fundModule(t, authtypes.FeeCollectorName, sdk.NewCoins(axpla(100)))
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(4)))

	events := eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "3", events[0][dynamicdeflationtypes.AttributeKeyStartHeight])
	require.Equal(t, "4", events[0][dynamicdeflationtypes.AttributeKeyEndHeight])
	require.Equal(t, "200", events[0][dynamicdeflationtypes.AttributeKeyGross])
	require.Equal(t, "40", events[0][dynamicdeflationtypes.AttributeKeyAllocated])

	feeCollectorBefore := fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom)
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(5)))
	require.Equal(t, feeCollectorBefore, fixture.moduleBalance(authtypes.FeeCollectorName, dynamicdeflationtypes.TargetDenom))
	require.Len(t, eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled), 1)
	hasPeriod, err := fixture.app.DynamicDeflationKeeper.CurrentPeriodStore.Has(fixture.ctx)
	require.NoError(t, err)
	require.False(t, hasPeriod)

	nextParams.Enabled = true
	fixture.setParams(t, nextParams)
	require.NoError(t, fixture.app.DynamicDeflationKeeper.BeginBlock(fixture.atHeight(6)))
	events = eventAttributesByType(fixture.ctx, dynamicdeflationtypes.EventTypeSettled)
	require.Len(t, events, 2)
	require.Equal(t, "6", events[1][dynamicdeflationtypes.AttributeKeyStartHeight])
	require.Equal(t, "6", events[1][dynamicdeflationtypes.AttributeKeyEndHeight])
	require.Equal(t, "80", events[1][dynamicdeflationtypes.AttributeKeyGross])
	require.Equal(t, "80", events[1][dynamicdeflationtypes.AttributeKeyAllocated])
}

type fixture struct {
	app *xplaapp.XplaApp
	ctx sdk.Context
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	app := xplaapp.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		t.TempDir(),
		xplaapp.EmptyAppOptions{},
		xplaapp.EmptyWasmOptions,
	)
	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{Time: time.Unix(1, 0).UTC()})
	require.NoError(t, app.MintKeeper.Params.Set(ctx, minttypes.DefaultParams()))
	app.RewardKeeper.InitGenesis(ctx, *rewardtypes.DefaultGenesisState())
	require.NoError(t, app.DynamicDeflationKeeper.InitGenesis(ctx, dynamicdeflationtypes.DefaultGenesisState()))
	require.NoError(t, app.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool()))
	return fixture{app: app, ctx: ctx}
}

func (f *fixture) atHeight(height int64) sdk.Context {
	f.ctx = f.ctx.WithBlockHeight(height)
	return f.ctx
}

func (f fixture) fundModule(t *testing.T, module string, coins sdk.Coins) {
	t.Helper()
	require.NoError(t, f.app.BankKeeper.MintCoins(f.ctx, minttypes.ModuleName, coins))
	require.NoError(t, f.app.BankKeeper.SendCoinsFromModuleToModule(f.ctx, minttypes.ModuleName, module, coins))
}

func (f fixture) fundCommunityPool(t *testing.T, coin sdk.Coin) {
	t.Helper()
	require.NoError(t, f.app.BankKeeper.MintCoins(f.ctx, minttypes.ModuleName, sdk.NewCoins(coin)))
	sender := f.app.AccountKeeper.GetModuleAddress(minttypes.ModuleName)
	require.NoError(t, f.app.DistrKeeper.FundCommunityPool(f.ctx, sdk.NewCoins(coin), sender))
}

func (f fixture) setParams(t *testing.T, params dynamicdeflationtypes.Params) {
	t.Helper()
	require.NoError(t, f.app.DynamicDeflationKeeper.SetParams(f.ctx, params))
}

func (f fixture) moduleBalance(module, denom string) sdkmath.Int {
	address := f.app.AccountKeeper.GetModuleAddress(module)
	return f.app.BankKeeper.GetBalance(f.ctx, address, denom).Amount
}

func (f fixture) communityPoolAmount(t *testing.T) sdkmath.Int {
	t.Helper()
	feePool, err := f.app.DistrKeeper.FeePool.Get(f.ctx)
	require.NoError(t, err)
	coins, _ := feePool.CommunityPool.TruncateDecimal()
	return coins.AmountOf(dynamicdeflationtypes.TargetDenom)
}

func paramsForSettlement(interval uint64, min, max int64) dynamicdeflationtypes.Params {
	params := dynamicdeflationtypes.DefaultParams()
	params.SettlementIntervalBlocks = interval
	params.MinFeeAmount = axpla(min)
	params.MaxFeeAmount = axpla(max)
	return params
}

func axpla(amount int64) sdk.Coin {
	return sdk.NewInt64Coin(dynamicdeflationtypes.TargetDenom, amount)
}

func eventAttributesByType(ctx sdk.Context, eventType string) []map[string]string {
	var matches []map[string]string
	for _, event := range ctx.EventManager().Events() {
		if event.Type != eventType {
			continue
		}
		attributes := make(map[string]string, len(event.Attributes))
		for _, attribute := range event.Attributes {
			attributes[attribute.Key] = attribute.Value
		}
		matches = append(matches, attributes)
	}
	return matches
}
