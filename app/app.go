package app

import (
	"fmt"
	"io"
	stdlog "log"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/server/api"
	"github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/version"
	authrest "github.com/cosmos/cosmos-sdk/x/auth/client/rest"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distrclient "github.com/cosmos/cosmos-sdk/x/distribution/client"
	govclient "github.com/cosmos/cosmos-sdk/x/gov/client"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramsclient "github.com/cosmos/cosmos-sdk/x/params/client"
	upgradeclient "github.com/cosmos/cosmos-sdk/x/upgrade/client"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibcclientclient "github.com/cosmos/ibc-go/v4/modules/core/02-client/client"
	"github.com/spf13/cast"
	abci "github.com/tendermint/tendermint/abci/types"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/libs/log"
	tmos "github.com/tendermint/tendermint/libs/os"
	dbm "github.com/tendermint/tm-db"
	volunteerclient "github.com/xpladev/xpla/x/volunteer/client"

	evmrest "github.com/evmos/ethermint/x/evm/client/rest"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmclient "github.com/CosmWasm/wasmd/x/wasm/client"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/prometheus/client_golang/prometheus"

	erc20client "github.com/evmos/evmos/v9/x/erc20/client"

	xplaante "github.com/xpladev/xpla/ante"
	"github.com/xpladev/xpla/app/keepers"
	"github.com/xpladev/xpla/app/openapiconsole"
	xplaappparams "github.com/xpladev/xpla/app/params"
	"github.com/xpladev/xpla/docs"

	evmupgrade "github.com/xpladev/xpla/app/upgrades/evm"
	"github.com/xpladev/xpla/app/upgrades/v1_4"
	"github.com/xpladev/xpla/app/upgrades/volunteer"
	xatpupgrade "github.com/xpladev/xpla/app/upgrades/xatp"
	xplareward "github.com/xpladev/xpla/app/upgrades/xpla_reward"
	xatpclient "github.com/xpladev/xpla/x/xatp/client"
)

var (
	// DefaultNodeHome default home directories for the application daemon
	DefaultNodeHome string
)

var (
	_ servertypes.Application = (*XplaApp)(nil)
)

var (
	// If EnabledSpecificProposals is "", and this is "true", then enable all x/wasm proposals.
	// If EnabledSpecificProposals is "", and this is not "true", then disable all x/wasm proposals.
	ProposalsEnabled = "true"
	// If set to non-empty string it must be comma-separated list of values that are all a subset
	// of "EnableAllProposals" (takes precedence over ProposalsEnabled)
	// https://github.com/CosmWasm/wasmd/blob/02a54d33ff2c064f3539ae12d75d027d9c665f05/x/wasm/internal/types/proposal.go#L28-L34
	EnableSpecificProposals = ""

	// TODO: after test, take this values from appOpts
	evmMaxGasWanted uint64 = 500000 //ethermintconfig.DefaultMaxTxGasWanted
)

// GetEnabledProposals parses the ProposalsEnabled / EnableSpecificProposals values to
// produce a list of enabled proposals to pass into wasmd app.
func GetEnabledProposals() []wasm.ProposalType {
	if EnableSpecificProposals == "" {
		if ProposalsEnabled == "true" {
			return wasm.EnableAllProposals
		}
		return wasm.DisableAllProposals
	}
	chunks := strings.Split(EnableSpecificProposals, ",")
	proposals, err := wasm.ConvertToProposals(chunks)
	if err != nil {
		panic(err)
	}
	return proposals
}

// GetWasmOpts build wasm options
func GetWasmOpts(appOpts servertypes.AppOptions) []wasm.Option {
	var wasmOpts []wasm.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	return wasmOpts
}

func getGovProposalHandlers() []govclient.ProposalHandler {
	var govProposalHandlers []govclient.ProposalHandler

	govProposalHandlers = wasmclient.ProposalHandlers

	govProposalHandlers = append(govProposalHandlers,
		paramsclient.ProposalHandler,
		distrclient.ProposalHandler,
		upgradeclient.ProposalHandler,
		upgradeclient.CancelProposalHandler,
		ibcclientclient.UpdateClientProposalHandler,
		ibcclientclient.UpgradeProposalHandler,
		erc20client.RegisterCoinProposalHandler, erc20client.RegisterERC20ProposalHandler, erc20client.ToggleTokenConversionProposalHandler,
	)

	govProposalHandlers = append(govProposalHandlers, volunteerclient.ProposalHandler...)
	govProposalHandlers = append(govProposalHandlers, xatpclient.ProposalHandler...)

	return govProposalHandlers
}

// XplaApp extends an ABCI application, but with most of its parameters exported.
// They are exported for convenience in creating helper functions, as object
// capabilities aren't needed for testing.
type XplaApp struct { // nolint: golint
	*baseapp.BaseApp
	keepers.AppKeepers

	legacyAmino       *codec.LegacyAmino
	appCodec          codec.Codec
	interfaceRegistry types.InterfaceRegistry

	invCheckPeriod uint

	// the module manager
	mm *module.Manager

	configurator module.Configurator
}

func init() {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		stdlog.Println("Failed to get home dir %2", err)
	}

	DefaultNodeHome = filepath.Join(userHomeDir, ".xpla")
	// apply custom power reduction for 'a' base denom unit 10^18
	sdk.DefaultPowerReduction = sdk.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

// NewXplaApp returns a reference to an initialized Xpla.
func NewXplaApp(
	logger log.Logger,
	db dbm.DB,
	traceStore io.Writer,
	loadLatest bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	encodingConfig xplaappparams.EncodingConfig,
	enabledProposals []wasm.ProposalType,
	appOpts servertypes.AppOptions,
	wasmOpts []wasm.Option,
	baseAppOptions ...func(*baseapp.BaseApp),
) *XplaApp {

	appCodec := encodingConfig.Codec
	legacyAmino := encodingConfig.Amino
	interfaceRegistry := encodingConfig.InterfaceRegistry

	bApp := baseapp.NewBaseApp(appName, logger, db, encodingConfig.TxConfig.TxDecoder(), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetVersion(version.Version)
	bApp.SetInterfaceRegistry(interfaceRegistry)

	app := &XplaApp{
		BaseApp:           bApp,
		legacyAmino:       legacyAmino,
		appCodec:          appCodec,
		interfaceRegistry: interfaceRegistry,
		invCheckPeriod:    invCheckPeriod,
	}

	moduleAccountAddresses := app.ModuleAccountAddrs()

	app.AppKeepers = keepers.NewAppKeeper(
		appCodec,
		bApp,
		legacyAmino,
		maccPerms,
		moduleAccountAddresses,
		app.BlockedModuleAccountAddrs(moduleAccountAddresses),
		skipUpgradeHeights,
		homePath,
		invCheckPeriod,
		enabledProposals,
		appOpts,
		wasmOpts,
	)

	skipGenesisInvariants := cast.ToBool(appOpts.Get(crisis.FlagSkipGenesisInvariants))

	// NOTE: Any module instantiated in the module manager that is later modified
	// must be passed by reference here.
	app.mm = module.NewManager(appModules(app, encodingConfig, skipGenesisInvariants)...)

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	// NOTE: staking module is required if HistoricalEntries param > 0
	// NOTE: capability module's beginblocker must come before any modules using capabilities (e.g. IBC)
	// Tell the app's module manager how to set the order of BeginBlockers, which are run at the beginning of every block.
	app.mm.SetOrderBeginBlockers(orderBeginBlockers()...)

	app.mm.SetOrderEndBlockers(orderEndBlockers()...)

	// NOTE: The genutils module must occur after staking so that pools are
	// properly initialized with tokens from genesis accounts.
	// NOTE: The genutils module must also occur after auth so that it can access the params from auth.
	// NOTE: Capability module must occur first so that it can initialize any capabilities
	// so that other modules that want to create or claim capabilities afterwards in InitChain
	// can do so safely.
	app.mm.SetOrderInitGenesis(orderInitBlockers()...)

	// Uncomment if you want to set a custom migration order here.
	// app.mm.SetOrderMigrations(custom order)

	app.mm.RegisterInvariants(&app.CrisisKeeper)
	app.mm.RegisterRoutes(app.Router(), app.QueryRouter(), encodingConfig.Amino)

	app.configurator = module.NewConfigurator(app.appCodec, app.MsgServiceRouter(), app.GRPCQueryRouter())
	app.mm.RegisterServices(app.configurator)

	// initialize stores
	app.MountKVStores(app.GetKVStoreKey())
	app.MountTransientStores(app.GetTransientStoreKey())
	app.MountMemoryStores(app.GetMemoryStoreKey())

	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	anteHandler, err := xplaante.NewAnteHandler(
		xplaante.HandlerOptions{
			AccountKeeper:        app.AccountKeeper,
			BankKeeper:           app.BankKeeper,
			EvmKeeper:            app.EvmKeeper,
			FeeMarketKeeper:      app.FeeMarketKeeper,
			FeegrantKeeper:       app.FeeGrantKeeper,
			VolunteerKeeper:      app.VolunteerKeeper,
			SignModeHandler:      encodingConfig.TxConfig.SignModeHandler(),
			SigGasConsumer:       xplaante.SigVerificationGasConsumer,
			IBCKeeper:            app.IBCKeeper,
			BypassMinFeeMsgTypes: cast.ToStringSlice(appOpts.Get(xplaappparams.BypassMinFeeMsgTypesKey)),
			TxCounterStoreKey:    app.GetKey(wasm.StoreKey),
			WasmConfig:           wasmConfig,
			MaxTxGasWanted:       evmMaxGasWanted,
			XATPKeeper:           app.XATPKeeper,
		},
	)
	if err != nil {
		panic(fmt.Errorf("failed to create AnteHandler: %s", err))
	}

	app.SetAnteHandler(anteHandler)
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetEndBlocker(app.EndBlocker)
	app.setUpgradeHandlers()

	if manager := app.SnapshotManager(); manager != nil {
		err = manager.RegisterExtensions(
			wasmkeeper.NewWasmSnapshotter(app.CommitMultiStore(), &app.WasmKeeper),
		)
		if err != nil {
			panic("failed to register snapshot extension: " + err.Error())
		}
	}

	if loadLatest {
		if err := app.LoadLatestVersion(); err != nil {
			tmos.Exit(fmt.Sprintf("failed to load latest version: %s", err))
		}
	}

	return app
}

// Name returns the name of the App
func (app *XplaApp) Name() string { return app.BaseApp.Name() }

// BeginBlocker application updates every begin block
func (app *XplaApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {
	return app.mm.BeginBlock(ctx, req)
}

// EndBlocker application updates every end block
func (app *XplaApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {
	return app.mm.EndBlock(ctx, req)
}

// InitChainer application update at chain initialization
func (app *XplaApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	if err := tmjson.Unmarshal(req.AppStateBytes, &genesisState); err != nil {
		panic(err)
	}

	app.UpgradeKeeper.SetModuleVersionMap(ctx, app.mm.GetVersionMap())

	return app.mm.InitGenesis(ctx, app.appCodec, genesisState)
}

// LoadHeight loads a particular height
func (app *XplaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height)
}

// ModuleAccountAddrs returns all the app's module account addresses.
func (app *XplaApp) ModuleAccountAddrs() map[string]bool {
	modAccAddrs := make(map[string]bool)
	for acc := range maccPerms {
		modAccAddrs[authtypes.NewModuleAddress(acc).String()] = true
	}

	return modAccAddrs
}

// BlockedModuleAccountAddrs returns all the app's blocked module account
// addresses.
func (app *XplaApp) BlockedModuleAccountAddrs(modAccAddrs map[string]bool) map[string]bool {
	// remove module accounts that are ALLOWED to received funds
	delete(modAccAddrs, authtypes.NewModuleAddress(govtypes.ModuleName).String())

	return modAccAddrs
}

// LegacyAmino returns XplaApp's amino codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *XplaApp) LegacyAmino() *codec.LegacyAmino {
	return app.legacyAmino
}

// AppCodec returns Xpla's app codec.
//
// NOTE: This is solely to be used for testing purposes as it may be desirable
// for modules to register their own custom testing types.
func (app *XplaApp) AppCodec() codec.Codec {
	return app.appCodec
}

// InterfaceRegistry returns Xpla's InterfaceRegistry
func (app *XplaApp) InterfaceRegistry() types.InterfaceRegistry {
	return app.interfaceRegistry
}

// RegisterAPIRoutes registers all application module routes with the provided
// API server.
func (app *XplaApp) RegisterAPIRoutes(apiSvr *api.Server, apiConfig config.APIConfig) {
	clientCtx := apiSvr.ClientCtx
	rpc.RegisterRoutes(clientCtx, apiSvr.Router)
	// Register legacy tx routes.
	authrest.RegisterTxRoutes(clientCtx, apiSvr.Router)
	// Register new tx routes from grpc-gateway.
	authtx.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)
	// Register new tendermint queries routes from grpc-gateway.
	tmservice.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// Register legacy and grpc-gateway routes for all modules.
	evmrest.RegisterTxRoutes(clientCtx, apiSvr.Router)
	ModuleBasics.RegisterRESTRoutes(clientCtx, apiSvr.Router)
	ModuleBasics.RegisterGRPCGatewayRoutes(clientCtx, apiSvr.GRPCGatewayRouter)

	// register app's OpenAPI routes.
	if apiConfig.Swagger {
		apiSvr.Router.Handle("/static/openapi.yml", http.FileServer(http.FS(docs.Docs)))
		apiSvr.Router.PathPrefix("/swagger/").Handler(openapiconsole.Handler(appName, "/static/openapi.yml"))
	}
}

// RegisterTxService implements the Application.RegisterTxService method.
func (app *XplaApp) RegisterTxService(clientCtx client.Context) {
	authtx.RegisterTxService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.BaseApp.Simulate, app.interfaceRegistry)
}

// RegisterTendermintService implements the Application.RegisterTendermintService method.
func (app *XplaApp) RegisterTendermintService(clientCtx client.Context) {
	tmservice.RegisterTendermintService(app.BaseApp.GRPCQueryRouter(), clientCtx, app.interfaceRegistry)
}

func (app *XplaApp) setUpgradeHandlers() {
	// XplaRewawrd upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		xplareward.UpgradeName,
		xplareward.CreateUpgradeHandler(app.mm, app.configurator, app.RewardKeeper),
	)

	// evm upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		evmupgrade.UpgradeName,
		evmupgrade.CreateUpgradeHandler(app.mm, app.configurator, *app.EvmKeeper, app.FeeMarketKeeper),
	)

	// Volunteer upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		volunteer.UpgradeName,
		volunteer.CreateUpgradeHandler(app.mm, app.configurator, &app.AppKeepers),
	)

	// v1_4 upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		v1_4.UpgradeName,
		v1_4.CreateUpgradeHandler(app.mm, app.configurator, app.AppKeepers.AccountKeeper, &app.AppKeepers.StakingKeeper),
	)

	// xatp upgrade handler
	app.UpgradeKeeper.SetUpgradeHandler(
		xatpupgrade.UpgradeName,
		xatpupgrade.CreateUpgradeHandler(app.mm, app.configurator, app.XATPKeeper, app.FeeMarketKeeper),
	)
	// When a planned update height is reached, the old binary will panic
	// writing on disk the height and name of the update that triggered it
	// This will read that value, and execute the preparations for the upgrade.
	upgradeInfo, err := app.UpgradeKeeper.ReadUpgradeInfoFromDisk()
	if err != nil {
		panic(fmt.Errorf("failed to read upgrade info from disk: %w", err))
	}

	if app.UpgradeKeeper.IsSkipHeight(upgradeInfo.Height) {
		return
	}

	var storeUpgrades *storetypes.StoreUpgrades

	switch upgradeInfo.Name {
	case xplareward.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{
			Added: xplareward.AddModules,
		}
	case evmupgrade.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{
			Added: evmupgrade.AddModules,
		}
	case volunteer.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{
			Added: volunteer.AddModules,
		}
	case v1_4.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{}

	case xatpupgrade.UpgradeName:
		storeUpgrades = &storetypes.StoreUpgrades{
			Added: xatpupgrade.AddModules,
		}
	}

	if storeUpgrades != nil {
		// configure store loader that checks if version == upgradeHeight and applies store upgrades
		app.SetStoreLoader(upgradetypes.UpgradeStoreLoader(upgradeInfo.Height, storeUpgrades))
	}
}
