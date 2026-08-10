package keeper

import (
	"context"
	"errors"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/dynamicdeflation/types"
)

type mockAccountKeeper struct{ addresses map[string]sdk.AccAddress }

func (m mockAccountKeeper) GetModuleAddress(name string) sdk.AccAddress { return m.addresses[name] }

type mockBankKeeper struct {
	accounts map[string]sdk.Coins
	modules  map[string]sdk.AccAddress
	failSend bool
	failBurn bool
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, m.accounts[addr.String()].AmountOf(denom))
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, from, to string, amount sdk.Coins) error {
	if m.failSend {
		return errors.New("send failed")
	}
	fromAddr, toAddr := m.modules[from], m.modules[to]
	if !m.accounts[fromAddr.String()].IsAllGTE(amount) {
		return errors.New("insufficient funds")
	}
	m.accounts[fromAddr.String()] = m.accounts[fromAddr.String()].Sub(amount...)
	m.accounts[toAddr.String()] = m.accounts[toAddr.String()].Add(amount...)
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, module string, amount sdk.Coins) error {
	if m.failBurn {
		return errors.New("burn failed")
	}
	addr := m.modules[module]
	if !m.accounts[addr.String()].IsAllGTE(amount) {
		return errors.New("insufficient funds")
	}
	m.accounts[addr.String()] = m.accounts[addr.String()].Sub(amount...)
	return nil
}

func (m *mockBankKeeper) set(module string, coins sdk.Coins) {
	m.accounts[m.modules[module].String()] = coins
}

func (m *mockBankKeeper) amount(module, denom string) int64 {
	return m.accounts[m.modules[module].String()].AmountOf(denom).Int64()
}

type mockDistributionKeeper struct {
	bank      *mockBankKeeper
	community sdk.Coins
	fail      bool
}

func (m *mockDistributionKeeper) FundCommunityPool(_ context.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if m.fail {
		return errors.New("community failed")
	}
	if !m.bank.accounts[sender.String()].IsAllGTE(amount) {
		return errors.New("insufficient funds")
	}
	m.bank.accounts[sender.String()] = m.bank.accounts[sender.String()].Sub(amount...)
	m.community = m.community.Add(amount...)
	return nil
}

type keeperFixture struct {
	keeper Keeper
	ctx    sdk.Context
	bank   *mockBankKeeper
	dist   *mockDistributionKeeper
}

func newKeeperFixture(t *testing.T) keeperFixture {
	t.Helper()
	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("dynamicdeflation_test")
	ctx := testutil.DefaultContext(key, tkey)
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	c := codec.NewProtoCodec(registry)
	modules := map[string]sdk.AccAddress{
		types.PoolName:             authtypes.NewModuleAddress(types.PoolName),
		authtypes.FeeCollectorName: authtypes.NewModuleAddress(authtypes.FeeCollectorName),
	}
	bank := &mockBankKeeper{accounts: map[string]sdk.Coins{}, modules: modules}
	for _, addr := range modules {
		bank.accounts[addr.String()] = sdk.NewCoins()
	}
	dist := &mockDistributionKeeper{bank: bank, community: sdk.NewCoins()}
	k := NewKeeper(c, runtime.NewKVStoreService(key), mockAccountKeeper{addresses: modules}, bank, dist, "authority")
	require.NoError(t, k.InitGenesis(ctx, types.DefaultGenesisState()))
	return keeperFixture{keeper: k, ctx: ctx, bank: bank, dist: dist}
}

func TestBeginBlockHeightOneNoOp(t *testing.T) {
	f := newKeeperFixture(t)
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(1)))
	require.Equal(t, int64(100), f.bank.amount(authtypes.FeeCollectorName, types.TargetDenom))
	has, err := f.keeper.CurrentPeriodStore.Has(f.ctx)
	require.NoError(t, err)
	require.False(t, has)
}

func TestBeginBlockRoutesOnlyTargetDenom(t *testing.T) {
	f := newKeeperFixture(t)
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100), sdk.NewInt64Coin("ufoo", 20)))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(2)))
	require.Equal(t, int64(80), f.bank.amount(authtypes.FeeCollectorName, types.TargetDenom))
	require.Equal(t, int64(20), f.bank.amount(authtypes.FeeCollectorName, "ufoo"))
	require.Equal(t, int64(20), f.bank.amount(types.PoolName, types.TargetDenom))
	period, err := f.keeper.CurrentPeriodStore.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, "100", period.GrossAmount.String())
	require.Equal(t, "20", period.AllocatedAmount.String())
}

func TestBeginBlockGrossOneSkipsTransfer(t *testing.T) {
	f := newKeeperFixture(t)
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 1)))
	f.bank.failSend = true
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(2)))
	period, err := f.keeper.CurrentPeriodStore.Get(f.ctx)
	require.NoError(t, err)
	require.True(t, period.GrossAmount.Equal(sdk.NewInt64Coin(types.TargetDenom, 1).Amount))
	require.True(t, period.AllocatedAmount.IsZero())
}

func TestBeginBlockFailsClosedOnPeriodAccumulatorOverflow(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.AllocationRate = sdkmath.LegacyZeroDec()
	p.SettlementIntervalBlocks = 2
	period := types.CurrentPeriod{
		StartHeight:     3,
		EndHeight:       4,
		ActiveConfig:    p,
		GrossAmount:     sdkmath.NewIntFromBigInt(new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))),
		AllocatedAmount: sdkmath.ZeroInt(),
	}
	require.NoError(t, f.keeper.CurrentPeriodStore.Set(f.ctx, period))
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 1)))

	err := f.keeper.BeginBlock(f.ctx.WithBlockHeight(4))
	require.ErrorContains(t, err, "accumulate period gross amount")
	stored, getErr := f.keeper.CurrentPeriodStore.Get(f.ctx)
	require.NoError(t, getErr)
	require.True(t, stored.GrossAmount.Equal(period.GrossAmount))
	require.Equal(t, int64(1), f.bank.amount(authtypes.FeeCollectorName, types.TargetDenom))
}

func TestBeginBlockPeriodBoundaryIncludesFinalBlock(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 3
	p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1000)
	p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 2000)
	require.NoError(t, f.keeper.SetParams(f.ctx, p))
	for height := int64(10); height <= 12; height++ {
		f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
		require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(height)))
		if height < 12 {
			require.Empty(t, eventAttributesByType(f.ctx, types.EventTypeSettled))
		} else {
			require.Len(t, eventAttributesByType(f.ctx, types.EventTypeSettled), 1)
		}
	}
	hasPeriod, err := f.keeper.CurrentPeriodStore.Has(f.ctx)
	require.NoError(t, err)
	require.False(t, hasPeriod)
	attributes := eventAttributesByType(f.ctx, types.EventTypeSettled)[0]
	require.Equal(t, "10", attributes[types.AttributeKeyStartHeight])
	require.Equal(t, "12", attributes[types.AttributeKeyEndHeight])
	require.Equal(t, "12", attributes[types.AttributeKeySettlementHeight])
	require.Equal(t, "300", attributes[types.AttributeKeyGross])
	require.Equal(t, "60", attributes[types.AttributeKeyAllocated])
	require.Equal(t, "60", attributes[types.AttributeKeyBurn])
	require.Equal(t, "0", attributes[types.AttributeKeyCommunity])
}

func TestBeginBlockSettlementAlignsToGlobalIntervalBoundary(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 1000
	p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1000)
	p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 2000)
	require.NoError(t, f.keeper.SetParams(f.ctx, p))
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))

	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(123000)))
	events := eventAttributesByType(f.ctx, types.EventTypeSettled)
	require.Len(t, events, 1)
	require.Equal(t, "123000", events[0][types.AttributeKeyStartHeight])
	require.Equal(t, "123000", events[0][types.AttributeKeyEndHeight])
	require.Equal(t, "123000", events[0][types.AttributeKeySettlementHeight])

	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(123001)))
	period, err := f.keeper.CurrentPeriodStore.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, int64(123001), period.StartHeight)
	require.Equal(t, int64(124000), period.EndHeight)
}

func TestBeginBlockSnapshotsParamsAndHonorsDisableAfterSettlement(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.SettlementIntervalBlocks = 2
	p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1000)
	p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 2000)
	require.NoError(t, f.keeper.SetParams(f.ctx, p))
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(5)))

	updated := p
	updated.Enabled = false
	updated.SettlementIntervalBlocks = 1
	updated.AllocationRate = types.DefaultAllocationRate.Add(types.DefaultAllocationRate)
	require.NoError(t, f.keeper.SetParams(f.ctx, updated))
	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(6)))
	attributes := eventAttributesByType(f.ctx, types.EventTypeSettled)
	require.Len(t, attributes, 1)
	require.Equal(t, "5", attributes[0][types.AttributeKeyStartHeight])
	require.Equal(t, "6", attributes[0][types.AttributeKeyEndHeight])
	require.Equal(t, "200", attributes[0][types.AttributeKeyGross])
	require.Equal(t, "40", attributes[0][types.AttributeKeyAllocated])

	f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(7)))
	require.Equal(t, int64(100), f.bank.amount(authtypes.FeeCollectorName, types.TargetDenom))
	has, err := f.keeper.CurrentPeriodStore.Has(f.ctx)
	require.NoError(t, err)
	require.False(t, has)
}

func TestBeginBlockReenableStartsFullNewPeriod(t *testing.T) {
	f := newKeeperFixture(t)
	p := types.DefaultParams()
	p.Enabled = false
	require.NoError(t, f.keeper.SetParams(f.ctx, p))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(20)))
	p.Enabled = true
	p.SettlementIntervalBlocks = 2
	require.NoError(t, f.keeper.SetParams(f.ctx, p))
	require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(21)))
	period, err := f.keeper.CurrentPeriodStore.Get(f.ctx)
	require.NoError(t, err)
	require.Equal(t, int64(21), period.StartHeight)
	require.Equal(t, int64(22), period.EndHeight)
}

func TestSettlementSurplusExcludedAndDeficitFailsClosed(t *testing.T) {
	t.Run("surplus", func(t *testing.T) {
		f := newKeeperFixture(t)
		p := types.DefaultParams()
		p.SettlementIntervalBlocks = 1
		p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1000)
		p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 2000)
		require.NoError(t, f.keeper.SetParams(f.ctx, p))
		f.bank.set(types.PoolName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 7), sdk.NewInt64Coin("ufoo", 9)))
		f.bank.set(authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 100)))
		require.NoError(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(2)))
		require.Equal(t, int64(7), f.bank.amount(types.PoolName, types.TargetDenom))
		require.Equal(t, int64(9), f.bank.amount(types.PoolName, "ufoo"))
	})

	t.Run("deficit", func(t *testing.T) {
		f := newKeeperFixture(t)
		p := types.DefaultParams()
		p.SettlementIntervalBlocks = 1
		period := types.CurrentPeriod{
			StartHeight: 2, EndHeight: 2, ActiveConfig: p,
			GrossAmount: sdk.NewInt64Coin(types.TargetDenom, 10).Amount, AllocatedAmount: sdk.NewInt64Coin(types.TargetDenom, 10).Amount,
		}
		require.NoError(t, f.keeper.CurrentPeriodStore.Set(f.ctx, period))
		f.bank.set(types.PoolName, sdk.NewCoins(sdk.NewInt64Coin(types.TargetDenom, 9)))
		require.ErrorContains(t, f.keeper.BeginBlock(f.ctx.WithBlockHeight(2)), "invariant violated")
		stored, err := f.keeper.CurrentPeriodStore.Get(f.ctx)
		require.NoError(t, err)
		require.Equal(t, period, stored)
		require.Empty(t, eventAttributesByType(f.ctx, types.EventTypeSettled))
	})
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
