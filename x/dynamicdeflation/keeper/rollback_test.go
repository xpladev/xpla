package keeper

import (
	"context"
	"errors"
	"fmt"
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

// storeBackedBankKeeper persists every observable bank mutation in the SDK
// multistore so the production CacheContext controls commit and rollback.
type storeBackedBankKeeper struct {
	key      *storetypes.KVStoreKey
	modules  map[string]sdk.AccAddress
	failBurn bool
}

func (m *storeBackedBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return sdk.NewCoin(denom, m.get(ctx, balanceKey(addr, denom)))
}

func (m *storeBackedBankKeeper) SendCoinsFromModuleToModule(ctx context.Context, from, to string, coins sdk.Coins) error {
	for _, coin := range coins {
		fromKey := balanceKey(m.modules[from], coin.Denom)
		toKey := balanceKey(m.modules[to], coin.Denom)
		fromAmount := m.get(ctx, fromKey)
		if fromAmount.LT(coin.Amount) {
			return errors.New("insufficient funds")
		}
		m.set(ctx, fromKey, fromAmount.Sub(coin.Amount))
		m.set(ctx, toKey, m.get(ctx, toKey).Add(coin.Amount))
	}
	return nil
}

func (m *storeBackedBankKeeper) BurnCoins(ctx context.Context, module string, coins sdk.Coins) error {
	for _, coin := range coins {
		accountKey := balanceKey(m.modules[module], coin.Denom)
		accountAmount := m.get(ctx, accountKey)
		supplyAmount := m.get(ctx, supplyKey(coin.Denom))
		if accountAmount.LT(coin.Amount) || supplyAmount.LT(coin.Amount) {
			return errors.New("insufficient funds")
		}
		// Mutate both balance and supply before injecting the error. The test
		// therefore proves that partial bank writes are discarded by CacheContext.
		m.set(ctx, accountKey, accountAmount.Sub(coin.Amount))
		m.set(ctx, supplyKey(coin.Denom), supplyAmount.Sub(coin.Amount))
	}
	if m.failBurn {
		return errors.New("injected burn failure after bank writes")
	}
	return nil
}

func (m *storeBackedBankKeeper) get(ctx context.Context, key []byte) sdkmath.Int {
	bz := sdk.UnwrapSDKContext(ctx).KVStore(m.key).Get(key)
	if len(bz) == 0 {
		return sdkmath.ZeroInt()
	}
	amount, ok := sdkmath.NewIntFromString(string(bz))
	if !ok {
		panic(fmt.Sprintf("invalid stored integer %q", bz))
	}
	return amount
}

func (m *storeBackedBankKeeper) set(ctx context.Context, key []byte, amount sdkmath.Int) {
	sdk.UnwrapSDKContext(ctx).KVStore(m.key).Set(key, []byte(amount.String()))
}

func balanceKey(addr sdk.AccAddress, denom string) []byte {
	return []byte("balance/" + addr.String() + "/" + denom)
}

func supplyKey(denom string) []byte { return []byte("supply/" + denom) }

func communityKey(denom string) []byte { return []byte("community/" + denom) }

type storeBackedDistributionKeeper struct {
	bank *storeBackedBankKeeper
	fail bool
}

func (m *storeBackedDistributionKeeper) FundCommunityPool(ctx context.Context, coins sdk.Coins, sender sdk.AccAddress) error {
	for _, coin := range coins {
		accountKey := balanceKey(sender, coin.Denom)
		accountAmount := m.bank.get(ctx, accountKey)
		if accountAmount.LT(coin.Amount) {
			return errors.New("insufficient funds")
		}
		// Model both the bank transfer and FeePool ledger update before failing.
		m.bank.set(ctx, accountKey, accountAmount.Sub(coin.Amount))
		m.bank.set(ctx, communityKey(coin.Denom), m.bank.get(ctx, communityKey(coin.Denom)).Add(coin.Amount))
	}
	if m.fail {
		return errors.New("injected community failure after bank and fee pool writes")
	}
	return nil
}

type rollbackFixture struct {
	keeper Keeper
	ctx    sdk.Context
	bank   *storeBackedBankKeeper
	dist   *storeBackedDistributionKeeper
}

func newRollbackFixture(t *testing.T) rollbackFixture {
	t.Helper()
	moduleKey := storetypes.NewKVStoreKey(types.StoreKey)
	bankKey := storetypes.NewKVStoreKey("dynamicdeflation_rollback_bank")
	ctx := testutil.DefaultContextWithKeys(
		map[string]*storetypes.KVStoreKey{types.StoreKey: moduleKey, bankKey.Name(): bankKey},
		nil,
		nil,
	)
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	c := codec.NewProtoCodec(registry)
	modules := map[string]sdk.AccAddress{
		types.PoolName:             authtypes.NewModuleAddress(types.PoolName),
		authtypes.FeeCollectorName: authtypes.NewModuleAddress(authtypes.FeeCollectorName),
	}
	bank := &storeBackedBankKeeper{key: bankKey, modules: modules}
	dist := &storeBackedDistributionKeeper{bank: bank}
	k := NewKeeper(c, runtime.NewKVStoreService(moduleKey), mockAccountKeeper{addresses: modules}, bank, dist, "authority")
	require.NoError(t, k.InitGenesis(ctx, types.DefaultGenesisState()))
	bank.set(ctx, balanceKey(modules[authtypes.FeeCollectorName], types.TargetDenom), sdkmath.NewInt(100))
	bank.set(ctx, balanceKey(modules[types.PoolName], types.TargetDenom), sdkmath.ZeroInt())
	bank.set(ctx, communityKey(types.TargetDenom), sdkmath.NewInt(7))
	bank.set(ctx, supplyKey(types.TargetDenom), sdkmath.NewInt(1000))
	return rollbackFixture{keeper: k, ctx: ctx, bank: bank, dist: dist}
}

func TestSettlementFailuresRollbackAllCachedState(t *testing.T) {
	tests := []struct {
		name      string
		configure func(*rollbackFixture, *types.Params)
	}{
		{
			name: "FundCommunityPool failure",
			configure: func(f *rollbackFixture, p *types.Params) {
				p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 0)
				p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1)
				f.dist.fail = true
			},
		},
		{
			name: "BurnCoins failure",
			configure: func(f *rollbackFixture, p *types.Params) {
				p.MinFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 1000)
				p.MaxFeeAmount = sdk.NewInt64Coin(types.TargetDenom, 2000)
				f.bank.failBurn = true
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newRollbackFixture(t)
			params := types.DefaultParams()
			params.SettlementIntervalBlocks = 1
			tt.configure(&f, &params)
			require.NoError(t, f.keeper.SetParams(f.ctx, params))

			feeCollector := f.bank.modules[authtypes.FeeCollectorName]
			poolAccount := f.bank.modules[types.PoolName]
			beforeFee := f.bank.get(f.ctx, balanceKey(feeCollector, types.TargetDenom))
			beforePool := f.bank.get(f.ctx, balanceKey(poolAccount, types.TargetDenom))
			beforeCommunity := f.bank.get(f.ctx, communityKey(types.TargetDenom))
			beforeSupply := f.bank.get(f.ctx, supplyKey(types.TargetDenom))
			beforeCurrent, err := f.keeper.CurrentPeriodStore.Has(f.ctx)
			require.NoError(t, err)

			err = f.keeper.BeginBlock(f.ctx.WithBlockHeight(2))
			require.Error(t, err)

			require.True(t, f.bank.get(f.ctx, balanceKey(feeCollector, types.TargetDenom)).Equal(beforeFee), "FeeCollector balance changed")
			require.True(t, f.bank.get(f.ctx, balanceKey(poolAccount, types.TargetDenom)).Equal(beforePool), "dynamic deflation pool balance changed")
			require.True(t, f.bank.get(f.ctx, communityKey(types.TargetDenom)).Equal(beforeCommunity), "community/FeePool ledger changed")
			require.True(t, f.bank.get(f.ctx, supplyKey(types.TargetDenom)).Equal(beforeSupply), "total supply changed")

			afterCurrent, err := f.keeper.CurrentPeriodStore.Has(f.ctx)
			require.NoError(t, err)
			require.Equal(t, beforeCurrent, afterCurrent, "CurrentPeriod presence changed")
			require.Empty(t, f.ctx.EventManager().Events(), "cached events leaked to parent context")
		})
	}
}
