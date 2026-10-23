package auth

import (
	"bytes"
	"context"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upstream "github.com/cosmos/cosmos-sdk/x/auth"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/require"
	xplaauthkeeper "github.com/xpladev/xpla/x/auth/keeper"
	googlegrpc "google.golang.org/grpc"
)

type authServiceCapture struct{ services map[string]any }

func (s *authServiceCapture) RegisterService(desc *googlegrpc.ServiceDesc, impl any) {
	if _, exists := s.services[desc.ServiceName]; exists {
		panic("duplicate service registration")
	}
	s.services[desc.ServiceName] = impl
}

type authMigrationCapture struct {
	module.Configurator
	names    []string
	versions []uint64
	handlers map[uint64]module.MigrationHandler
}

func (c *authMigrationCapture) RegisterMigration(name string, version uint64, handler module.MigrationHandler) error {
	c.names = append(c.names, name)
	c.versions = append(c.versions, version)
	if c.handlers == nil {
		c.handlers = map[uint64]module.MigrationHandler{}
	}
	c.handlers[version] = handler
	return c.Configurator.RegisterMigration(name, version, handler)
}

func TestAuthModuleRegistrationMatchesSDK(t *testing.T) {
	key := storetypes.NewKVStoreKey(authtypes.StoreKey)
	registry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	keeper := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(key), authtypes.ProtoBaseAccount,
		map[string][]string{}, authcodec.NewBech32Codec(sdk.Bech32MainPrefix), sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress("gov").String())
	capture := func(am module.HasServices) (*authServiceCapture, *authServiceCapture, *authMigrationCapture) {
		msg := &authServiceCapture{services: map[string]any{}}
		query := &authServiceCapture{services: map[string]any{}}
		cfg := &authMigrationCapture{Configurator: module.NewConfigurator(cdc, msg, query)}
		am.RegisterServices(cfg)
		require.NoError(t, cfg.Error())
		return msg, query, cfg
	}
	upstreamModule := upstream.NewAppModule(cdc, keeper, nil, nil)
	localModule := NewAppModule(cdc, xplaauthkeeper.NewAccountKeeper(keeper), nil, nil, nil)
	sdkMsg, sdkQuery, sdkCfg := capture(upstreamModule)
	localMsg, localQuery, localCfg := capture(localModule)
	require.Equal(t, upstreamModule.ConsensusVersion(), localModule.ConsensusVersion())
	require.Equal(t, sdkCfg.names, localCfg.names)
	require.Equal(t, sdkCfg.versions, localCfg.versions)
	require.Equal(t, []uint64{1, 2, 3, 4}, localCfg.versions)
	require.Len(t, localCfg.handlers, len(localCfg.versions))
	require.Len(t, localMsg.services, 1)
	require.Len(t, localQuery.services, 1)
	for name, impl := range sdkMsg.services {
		require.Contains(t, localMsg.services, name)
		require.IsType(t, impl, localMsg.services[name])
		require.Implements(t, (*authtypes.MsgServer)(nil), localMsg.services[name])
	}
	for name := range sdkQuery.services {
		require.Contains(t, localQuery.services, name)
		require.IsType(t, xplaauthkeeper.NewQueryServer(xplaauthkeeper.NewAccountKeeper(keeper), nil), localQuery.services[name])
		require.Implements(t, (*authtypes.QueryServer)(nil), localQuery.services[name])
	}
}

// The SDK auth 1->2 migrator requires the original concrete query router.
// Executing its captured handler checks more than migration registration alone.
func TestAuthQueryModulePreservesMigrationRouter(t *testing.T) {
	key := storetypes.NewKVStoreKey(authtypes.StoreKey)
	ctx := testutil.DefaultContext(key, storetypes.NewTransientStoreKey("auth_migration_test"))
	registry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(registry)
	stakingtypes.RegisterInterfaces(registry)
	cdc := codec.NewProtoCodec(registry)
	keeper := authkeeper.NewAccountKeeper(cdc, runtime.NewKVStoreService(key), authtypes.ProtoBaseAccount,
		map[string][]string{}, authcodec.NewBech32Codec(sdk.Bech32MainPrefix), sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress("gov").String())
	account := keeper.NewAccountWithAddress(ctx, sdk.AccAddress(bytes.Repeat([]byte{0x12}, 20)))
	keeper.SetAccount(ctx, account)
	router := baseapp.NewGRPCQueryRouter()
	router.SetInterfaceRegistry(registry)
	staking := &migrationStakingQueryServer{}
	stakingtypes.RegisterQueryServer(router, staking)
	msg := &authServiceCapture{services: map[string]any{}}
	cfg := &authMigrationCapture{Configurator: module.NewConfigurator(cdc, msg, router)}
	NewAppModule(cdc, xplaauthkeeper.NewAccountKeeper(keeper), nil, nil, nil).RegisterServices(cfg)
	require.NoError(t, cfg.Error())
	require.NotNil(t, cfg.handlers[1])
	require.NoError(t, cfg.handlers[1](ctx))
	require.Equal(t, 1, staking.paramsCalls)
	require.Equal(t, account, keeper.GetAccount(ctx, account.GetAddress()))
}

type migrationStakingQueryServer struct {
	stakingtypes.QueryServer
	paramsCalls int
}

func (s *migrationStakingQueryServer) Params(context.Context, *stakingtypes.QueryParamsRequest) (*stakingtypes.QueryParamsResponse, error) {
	s.paramsCalls++
	return &stakingtypes.QueryParamsResponse{Params: stakingtypes.DefaultParams()}, nil
}
