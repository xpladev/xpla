package v1_11_test

import (
	"testing"

	"cosmossdk.io/log/v2"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/evm/x/vm/types"
	pfmrouter "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware"
	pfmlegacy "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/migrations/v4/legacy"
	pfmtypes "github.com/cosmos/ibc-go/v11/modules/apps/packet-forward-middleware/types"
	ratelimit "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting"
	ratelimittypes "github.com/cosmos/ibc-go/v11/modules/apps/rate-limiting/types"

	xplaapp "github.com/xpladev/xpla/app"
	v1_11 "github.com/xpladev/xpla/app/upgrades/v1_11"
)

func TestIBCMiddlewareStoresReuseV1_10Keys(t *testing.T) {
	require.Equal(t, "packetfowardmiddleware", pfmtypes.StoreKey)
	require.Equal(t, "ratelimit", ratelimittypes.StoreKey)
	require.Empty(t, v1_11.StoreUpgrades.Added)
	require.Empty(t, v1_11.StoreUpgrades.Renamed)
	require.Empty(t, v1_11.StoreUpgrades.Deleted)
}

func TestUpgradeHandlerMigratesIBCMiddlewareState(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	pfmStore := ctx.KVStore(xpla.GetKey(pfmtypes.StoreKey))
	pfmKey := setLegacyPFMPacket(t, pfmStore, false)

	rateLimitStore := ctx.KVStore(xpla.GetKey(ratelimittypes.StoreKey))
	pendingSendKey := prefixedKey(ratelimittypes.PendingSendPacketPrefix, "legacy-send")
	pendingReceiveKey := prefixedKey(ratelimittypes.PendingReceivePacketPrefix, "legacy-receive")
	rateLimitKey := prefixedKey(ratelimittypes.RateLimitKeyPrefix, "rate-limit-state")
	rateLimitStore.Set(pendingSendKey, []byte("pending-send"))
	rateLimitStore.Set(pendingReceiveKey, []byte("pending-receive"))
	rateLimitStore.Set(rateLimitKey, []byte("preserved"))

	mm, configurator := newIBCMigrationManager(t, xpla)
	handler := v1_11.CreateUpgradeHandler(mm, configurator, &xpla.AppKeepers, xpla.AppCodec())
	vm, err := handler(ctx, upgradetypes.Plan{}, module.VersionMap{
		pfmtypes.ModuleName:       3,
		ratelimittypes.ModuleName: 1,
	})
	require.NoError(t, err)
	require.EqualValues(t, 4, vm[pfmtypes.ModuleName])
	require.EqualValues(t, 2, vm[ratelimittypes.ModuleName])

	var migratedPacket pfmtypes.InFlightPacket
	require.NoError(t, migratedPacket.Unmarshal(pfmStore.Get(pfmKey)))
	require.Equal(t, "channel-1", migratedPacket.PacketSrcChannelId)
	require.Equal(t, "transfer", migratedPacket.PacketSrcPortId)
	require.Nil(t, rateLimitStore.Get(pendingSendKey))
	require.Nil(t, rateLimitStore.Get(pendingReceiveKey))
	require.Equal(t, []byte("preserved"), rateLimitStore.Get(rateLimitKey))
}

func TestUpgradeHandlerRejectsNonrefundablePFMPacket(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	pfmStore := ctx.KVStore(xpla.GetKey(pfmtypes.StoreKey))
	pfmKey := setLegacyPFMPacket(t, pfmStore, true)

	mm, configurator := newIBCMigrationManager(t, xpla)
	handler := v1_11.CreateUpgradeHandler(mm, configurator, &xpla.AppKeepers, xpla.AppCodec())
	_, err := handler(ctx, upgradetypes.Plan{}, module.VersionMap{
		pfmtypes.ModuleName:       3,
		ratelimittypes.ModuleName: 1,
	})
	require.ErrorContains(t, err, "nonrefundable in-flight packet")
	require.ErrorContains(t, err, string(pfmKey))
}

func TestUpgradeHandlerDoesNotMutateEVMState(t *testing.T) {
	xpla, ctx := setupUpgradeState(t)

	params := types.DefaultParams()
	params.EvmDenom = "axpla"
	params.ExtendedDenomOptions = nil
	params.HistoryServeWindow = 0
	params.ActiveStaticPrecompiles = []string{types.BankPrecompileAddress}
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, params))
	paramsBeforeUpgrade := xpla.EvmKeeper.GetParams(ctx)

	handler := v1_11.CreateUpgradeHandler(
		module.NewManager(),
		module.NewConfigurator(nil, nil, nil),
		&xpla.AppKeepers,
		nil,
	)

	_, err := handler(ctx, upgradetypes.Plan{}, module.VersionMap{})
	require.NoError(t, err)
	require.Equal(t, paramsBeforeUpgrade, xpla.EvmKeeper.GetParams(ctx))
}

func setupUpgradeState(t *testing.T) (*xplaapp.XplaApp, sdk.Context) {
	t.Helper()

	xpla := xplaapp.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		true,
		map[int64]bool{},
		t.TempDir(),
		xplaapp.EmptyAppOptions{},
		xplaapp.EmptyWasmOptions,
	)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{Height: 1})

	return xpla, ctx
}

func newIBCMigrationManager(t *testing.T, xpla *xplaapp.XplaApp) (*module.Manager, module.Configurator) {
	t.Helper()

	msgRouter := baseapp.NewMsgServiceRouter()
	msgRouter.SetInterfaceRegistry(xpla.InterfaceRegistry())
	queryRouter := baseapp.NewGRPCQueryRouter()
	queryRouter.SetInterfaceRegistry(xpla.InterfaceRegistry())

	mm := module.NewManager(
		pfmrouter.NewAppModule(xpla.PFMRouterKeeper),
		ratelimit.NewAppModule(xpla.RatelimitKeeper),
	)
	configurator := module.NewConfigurator(xpla.AppCodec(), msgRouter, queryRouter)
	require.NoError(t, mm.RegisterServices(configurator))

	return mm, configurator
}

func setLegacyPFMPacket(t *testing.T, store interface {
	Get([]byte) []byte
	Set([]byte, []byte)
}, nonrefundable bool) []byte {
	t.Helper()

	packet := pfmlegacy.InFlightPacket{
		PacketData:             []byte{1, 2, 3},
		OriginalSenderAddress:  "xpla1legacy",
		RefundChannelId:        "channel-refund",
		RefundPortId:           "transfer",
		PacketSrcChannelId:     "channel-1",
		PacketSrcPortId:        "transfer",
		PacketTimeoutTimestamp: 100,
		PacketTimeoutHeight:    "0-10",
		RefundSequence:         1,
		RetriesRemaining:       1,
		Timeout:                1_000,
		Nonrefundable:          nonrefundable,
	}
	bz, err := packet.Marshal()
	require.NoError(t, err)

	key := pfmtypes.RefundPacketKey(packet.PacketSrcChannelId, packet.PacketSrcPortId, packet.RefundSequence)
	store.Set(key, bz)
	require.Equal(t, bz, store.Get(key))

	return key
}

func prefixedKey(prefix []byte, suffix string) []byte {
	return append(append([]byte(nil), prefix...), suffix...)
}
