package app

import (
	"bytes"
	"testing"

	"cosmossdk.io/log"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
	xplaauth "github.com/xpladev/xpla/x/auth"
	xplawasm "github.com/xpladev/xpla/x/wasm"
	googlegrpc "google.golang.org/grpc"
)

func TestAuthQueriesUseWasmRegistryThroughAppRouter(t *testing.T) {
	app := NewXplaApp(log.NewNopLogger(), dbm.NewMemDB(), nil, true, map[int64]bool{}, t.TempDir(),
		EmptyAppOptions{}, EmptyWasmOptions, baseapp.SetChainID("auth-alias-query-test"))
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	require.IsType(t, xplaauth.AppModule{}, app.mm.Modules[authtypes.ModuleName])
	authSimulationModules := 0
	for _, simulationModule := range app.SimulationManager().Modules {
		named, ok := simulationModule.(interface{ Name() string })
		if ok && named.Name() == authtypes.ModuleName {
			require.IsType(t, xplaauth.AppModule{}, simulationModule)
			authSimulationModules++
		}
	}
	require.Equal(t, 1, authSimulationModules)
	t.Run("Wasm migration registration", func(t *testing.T) {
		wasmModule, ok := app.mm.Modules[wasmtypes.ModuleName].(xplawasm.AppModule)
		require.True(t, ok)
		require.Equal(t, uint64(4), wasmModule.ConsensusVersion())
		cfg := &migrationRegistrationCapture{
			Configurator: module.NewConfigurator(app.AppCodec(), googlegrpc.NewServer(), googlegrpc.NewServer()),
			versions:     map[string][]uint64{},
		}
		wasmModule.RegisterServices(cfg)
		require.NoError(t, cfg.Error())
		require.Equal(t, map[string][]uint64{wasmtypes.ModuleName: {1, 2, 3}}, cfg.versions)
	})
	ctx := app.NewUncachedContext(false, tmproto.Header{ChainID: "auth-alias-query-test"})
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	contract := sdk.AccAddress(append(bytes.Repeat([]byte{0x32}, 12), alias...))
	contractAccount := app.AccountKeeper.NewAccountWithAddress(ctx, contract)
	require.NoError(t, contractAccount.SetSequence(7))
	app.AccountKeeper.SetAccount(ctx, contractAccount)
	require.NoError(t, app.WasmKeeper.RegisterWasmAlias(ctx, alias, contract))
	require.Nil(t, app.AccountKeeper.GetAccount(ctx, alias), "SDK identity lookup must remain exact")
	requested, err := app.AccountKeeper.AddressCodec().BytesToString(alias)
	require.NoError(t, err)

	query := func(method string, request, response proto.Message) {
		t.Helper()
		path := "/cosmos.auth.v1beta1.Query/" + method
		handler := app.GRPCQueryRouter().Route(path)
		require.NotNil(t, handler)
		result, err := handler(ctx, &abci.RequestQuery{Path: path, Data: app.AppCodec().MustMarshal(request)})
		require.NoError(t, err)
		require.Zero(t, result.Code)
		require.NoError(t, app.AppCodec().Unmarshal(result.Value, response))
	}
	for _, exact := range []bool{false, true} {
		expectedAddress, expectedSequence := contract, uint64(7)
		if exact {
			account := app.AccountKeeper.NewAccountWithAddress(ctx, alias)
			require.NoError(t, account.SetSequence(18))
			app.AccountKeeper.SetAccount(ctx, account)
			expectedAddress, expectedSequence = alias, 18
		}
		var accountResponse authtypes.QueryAccountResponse
		query("Account", &authtypes.QueryAccountRequest{Address: requested}, &accountResponse)
		var account sdk.AccountI
		require.NoError(t, app.AppCodec().UnpackAny(accountResponse.Account, &account))
		require.Equal(t, expectedAddress, account.GetAddress())
		require.Equal(t, expectedSequence, account.GetSequence())
		var infoResponse authtypes.QueryAccountInfoResponse
		query("AccountInfo", &authtypes.QueryAccountInfoRequest{Address: requested}, &infoResponse)
		require.Equal(t, requested, infoResponse.Info.Address)
		require.Equal(t, expectedSequence, infoResponse.Info.Sequence)
	}
}

type migrationRegistrationCapture struct {
	module.Configurator
	versions map[string][]uint64
}

func (c *migrationRegistrationCapture) RegisterMigration(moduleName string, fromVersion uint64, handler module.MigrationHandler) error {
	c.versions[moduleName] = append(c.versions[moduleName], fromVersion)
	return c.Configurator.RegisterMigration(moduleName, fromVersion, handler)
}
