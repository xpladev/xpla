package keepers

import (
	"path/filepath"
	"strings"

	wasmapp "github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/store/streaming"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	authzkeeper "github.com/cosmos/cosmos-sdk/x/authz/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	crisiskeeper "github.com/cosmos/cosmos-sdk/x/crisis/keeper"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	evidencekeeper "github.com/cosmos/cosmos-sdk/x/evidence/keeper"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	feegrantkeeper "github.com/cosmos/cosmos-sdk/x/feegrant/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	"github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	paramproposal "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/cosmos-sdk/x/upgrade"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router"
	routerkeeper "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router/keeper"
	routertypes "github.com/cosmos/ibc-apps/middleware/packet-forward-middleware/v4/router/types"
	icacontroller "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller"
	icacontrollerkeeper "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/keeper"
	icacontrollertypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/controller/types"
	icahost "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host"
	icahostkeeper "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/keeper"
	icahosttypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/types"
	ibcfee "github.com/cosmos/ibc-go/v4/modules/apps/29-fee"
	ibcfeekeeper "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/keeper"
	ibcfeetypes "github.com/cosmos/ibc-go/v4/modules/apps/29-fee/types"
	"github.com/cosmos/ibc-go/v4/modules/apps/transfer"
	ibctransferkeeper "github.com/cosmos/ibc-go/v4/modules/apps/transfer/keeper"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v4/modules/core/02-client"
	ibcclienttypes "github.com/cosmos/ibc-go/v4/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibchost "github.com/cosmos/ibc-go/v4/modules/core/24-host"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/keeper"
	etherminttypes "github.com/evmos/ethermint/types"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
	feemarkettypes "github.com/evmos/ethermint/x/feemarket/types"
	"github.com/evmos/evmos/v9/x/erc20"
	erc20keeper "github.com/evmos/evmos/v9/x/erc20/keeper"
	erc20types "github.com/evmos/evmos/v9/x/erc20/types"
	tmos "github.com/tendermint/tendermint/libs/os"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	xplastakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
	volunteerkeeper "github.com/xpladev/xpla/x/volunteer/keeper"
	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"
)

type AppKeepers struct {
	// keys to access the substores
	keys    map[string]*storetypes.KVStoreKey
	tkeys   map[string]*storetypes.TransientStoreKey
	memKeys map[string]*storetypes.MemoryStoreKey

	// keepers
	AccountKeeper    authkeeper.AccountKeeper
	BankKeeper       bankkeeper.Keeper
	CapabilityKeeper *capabilitykeeper.Keeper
	StakingKeeper    xplastakingkeeper.Keeper
	SlashingKeeper   slashingkeeper.Keeper
	MintKeeper       mintkeeper.Keeper
	DistrKeeper      distrkeeper.Keeper
	GovKeeper        govkeeper.Keeper
	CrisisKeeper     crisiskeeper.Keeper
	UpgradeKeeper    upgradekeeper.Keeper
	ParamsKeeper     paramskeeper.Keeper
	// IBC Keeper must be a pointer in the app, so we can SetRouter on it correctly
	IBCKeeper           *ibckeeper.Keeper
	IBCFeeKeeper        ibcfeekeeper.Keeper
	ICAControllerKeeper icacontrollerkeeper.Keeper
	ICAHostKeeper       icahostkeeper.Keeper
	EvidenceKeeper      evidencekeeper.Keeper
	TransferKeeper      ibctransferkeeper.Keeper
	FeeGrantKeeper      feegrantkeeper.Keeper
	AuthzKeeper         authzkeeper.Keeper
	RouterKeeper        *routerkeeper.Keeper
	RewardKeeper        rewardkeeper.Keeper
	VolunteerKeeper     volunteerkeeper.Keeper

	// make scoped keepers public for test purposes
	ScopedIBCKeeper           capabilitykeeper.ScopedKeeper
	ScopedICAControllerKeeper capabilitykeeper.ScopedKeeper
	ScopedFeeMockKeeper       capabilitykeeper.ScopedKeeper
	ScopedTransferKeeper      capabilitykeeper.ScopedKeeper
	ScopedICAHostKeeper       capabilitykeeper.ScopedKeeper

	WasmKeeper       wasm.Keeper
	scopedWasmKeeper capabilitykeeper.ScopedKeeper

	EvmKeeper       *evmkeeper.Keeper
	FeeMarketKeeper feemarketkeeper.Keeper
	Erc20Keeper     erc20keeper.Keeper
}

var (
	// TODO: after test, take this values from appOpts
	evmTrace = "" //ethermintconfig.DefaultEVMTracer,
)

func NewAppKeeper(
	appCodec codec.Codec,
	bApp *baseapp.BaseApp,
	legacyAmino *codec.LegacyAmino,
	maccPerms map[string][]string,
	modAccAddrs map[string]bool,
	blockedAddress map[string]bool,
	skipUpgradeHeights map[int64]bool,
	homePath string,
	invCheckPeriod uint,
	enabledProposals []wasm.ProposalType,
	appOpts servertypes.AppOptions,
	wasmOpts []wasm.Option,
) AppKeepers {
	appKeepers := AppKeepers{}

	// Set keys KVStoreKey, TransientStoreKey, MemoryStoreKey
	appKeepers.GenerateKeys()

	// configure state listening capabilities using AppOptions
	// we are doing nothing with the returned streamingServices and waitGroup in this case
	if _, _, err := streaming.LoadStreamingServices(bApp, appOpts, appCodec, appKeepers.keys); err != nil {
		tmos.Exit(err.Error())
	}

	appKeepers.ParamsKeeper = initParamsKeeper(
		appCodec,
		legacyAmino,
		appKeepers.keys[paramstypes.StoreKey],
		appKeepers.tkeys[paramstypes.TStoreKey],
	)

	// set the BaseApp's parameter store
	bApp.SetParamStore(
		appKeepers.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramskeeper.ConsensusParamsKeyTable()),
	)

	// add capability keeper and ScopeToModule for ibc module
	appKeepers.CapabilityKeeper = capabilitykeeper.NewKeeper(appCodec, appKeepers.keys[capabilitytypes.StoreKey], appKeepers.memKeys[capabilitytypes.MemStoreKey])
	appKeepers.ScopedIBCKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibchost.ModuleName)
	appKeepers.ScopedICAControllerKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icacontrollertypes.SubModuleName)
	appKeepers.ScopedTransferKeeper = appKeepers.CapabilityKeeper.ScopeToModule(ibctransfertypes.ModuleName)
	appKeepers.ScopedICAHostKeeper = appKeepers.CapabilityKeeper.ScopeToModule(icahosttypes.SubModuleName)
	appKeepers.scopedWasmKeeper = appKeepers.CapabilityKeeper.ScopeToModule(wasm.ModuleName)

	appKeepers.CapabilityKeeper.Seal()

	appKeepers.CrisisKeeper = crisiskeeper.NewKeeper(
		appKeepers.GetSubspace(crisistypes.ModuleName),
		invCheckPeriod,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
	)

	// add normal keepers
	appKeepers.AccountKeeper = authkeeper.NewAccountKeeper(
		appCodec,
		appKeepers.keys[authtypes.StoreKey],
		appKeepers.GetSubspace(authtypes.ModuleName),
		etherminttypes.ProtoAccount,
		maccPerms,
	)
	appKeepers.BankKeeper = bankkeeper.NewBaseKeeper(
		appCodec,
		appKeepers.keys[banktypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.GetSubspace(banktypes.ModuleName),
		blockedAddress,
	)
	appKeepers.AuthzKeeper = authzkeeper.NewKeeper(
		appKeepers.keys[authzkeeper.StoreKey],
		appCodec,
		bApp.MsgServiceRouter(),
	)
	appKeepers.FeeGrantKeeper = feegrantkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[feegrant.StoreKey],
		appKeepers.AccountKeeper,
	)
	stakingKeeper := xplastakingkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[stakingtypes.StoreKey],
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.GetSubspace(stakingtypes.ModuleName),
		&appKeepers.VolunteerKeeper,
	)
	appKeepers.MintKeeper = mintkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[minttypes.StoreKey],
		appKeepers.GetSubspace(minttypes.ModuleName),
		&appKeepers.StakingKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		authtypes.FeeCollectorName,
	)
	appKeepers.DistrKeeper = distrkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[distrtypes.StoreKey],
		appKeepers.GetSubspace(distrtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		&appKeepers.StakingKeeper,
		authtypes.FeeCollectorName,
		modAccAddrs,
	)
	appKeepers.SlashingKeeper = slashingkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[slashingtypes.StoreKey],
		&appKeepers.StakingKeeper,
		appKeepers.GetSubspace(slashingtypes.ModuleName),
	)

	appKeepers.VolunteerKeeper = volunteerkeeper.NewKeeper(
		appKeepers.keys[volunteertypes.StoreKey],
		appCodec,
		&appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
	)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	appKeepers.StakingKeeper = stakingKeeper
	appKeepers.StakingKeeper.Keeper.SetHooks(
		stakingtypes.NewMultiStakingHooks(appKeepers.DistrKeeper.Hooks(), appKeepers.SlashingKeeper.Hooks()),
	)

	appKeepers.UpgradeKeeper = upgradekeeper.NewKeeper(
		skipUpgradeHeights,
		appKeepers.keys[upgradetypes.StoreKey],
		appCodec,
		homePath,
		bApp,
	)

	// ... other modules keeper

	appKeepers.IBCKeeper = ibckeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibchost.StoreKey],
		appKeepers.GetSubspace(ibchost.ModuleName),
		appKeepers.StakingKeeper,
		appKeepers.UpgradeKeeper,
		appKeepers.ScopedIBCKeeper,
	)

	wasmDir := filepath.Join(homePath, "data")
	wasmConfig, err := wasm.ReadWasmConfig(appOpts)
	if err != nil {
		panic("error while reading wasm config: " + err.Error())
	}

	availableCapabilities := strings.Join(wasmapp.AllCapabilities(), ",")

	// Stargate Queries
	accepted := wasmkeeper.AcceptedStargateQueries{
		// ibc
		"/ibc.core.client.v1.Query/ClientState":    &ibcclienttypes.QueryClientStateResponse{},
		"/ibc.core.client.v1.Query/ConsensusState": &ibcclienttypes.QueryConsensusStateResponse{},
		"/ibc.core.connection.v1.Query/Connection": &ibcconnectiontypes.QueryConnectionResponse{},

		// governance
		"/cosmos.gov.v1beta1.Query/Vote": &govtypes.QueryVoteResponse{},

		// staking
		"/cosmos.staking.v1beta1.Query/Delegation":          &stakingtypes.QueryDelegationResponse{},
		"/cosmos.staking.v1beta1.Query/Redelegations":       &stakingtypes.QueryRedelegationsResponse{},
		"/cosmos.staking.v1beta1.Query/UnbondingDelegation": &stakingtypes.QueryUnbondingDelegationResponse{},
		"/cosmos.staking.v1beta1.Query/Validator":           &stakingtypes.QueryValidatorResponse{},
	}
	querierOpts := wasmkeeper.WithQueryPlugins(
		&wasmkeeper.QueryPlugins{
			Stargate: wasmkeeper.AcceptListStargateQuerier(accepted, bApp.GRPCQueryRouter(), appCodec),
		})
	wasmOpts = append(wasmOpts, querierOpts)
	appKeepers.WasmKeeper = wasm.NewKeeper(
		appCodec,
		appKeepers.keys[wasm.StoreKey],
		appKeepers.GetSubspace(wasm.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
		appKeepers.IBCFeeKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.scopedWasmKeeper,
		appKeepers.TransferKeeper,
		bApp.MsgServiceRouter(),
		bApp.GRPCQueryRouter(),
		wasmDir,
		wasmConfig,
		availableCapabilities,
		wasmOpts...,
	)

	// register the proposal types
	govRouter := govtypes.NewRouter()
	govRouter.
		AddRoute(govtypes.RouterKey, govtypes.ProposalHandler).
		AddRoute(paramproposal.RouterKey, params.NewParamChangeProposalHandler(appKeepers.ParamsKeeper)).
		AddRoute(distrtypes.RouterKey, distr.NewCommunityPoolSpendProposalHandler(appKeepers.DistrKeeper)).
		AddRoute(upgradetypes.RouterKey, upgrade.NewSoftwareUpgradeProposalHandler(appKeepers.UpgradeKeeper)).
		AddRoute(ibcclienttypes.RouterKey, ibcclient.NewClientProposalHandler(appKeepers.IBCKeeper.ClientKeeper)).
		AddRoute(erc20types.RouterKey, erc20.NewErc20ProposalHandler(&appKeepers.Erc20Keeper)).
		AddRoute(volunteertypes.RouterKey, volunteerkeeper.NewVolunteerValidatorProposalHandler(appKeepers.VolunteerKeeper))

	// register wasm gov proposal types
	if len(enabledProposals) != 0 {
		govRouter.AddRoute(wasm.RouterKey, wasm.NewWasmProposalHandler(appKeepers.WasmKeeper, enabledProposals))
	}

	appKeepers.GovKeeper = govkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[govtypes.StoreKey],
		appKeepers.GetSubspace(govtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		&stakingKeeper,
		govRouter,
	)

	// Create Ethermint keepers
	appKeepers.FeeMarketKeeper = feemarketkeeper.NewKeeper(
		appCodec,
		appKeepers.GetSubspace(feemarkettypes.ModuleName),
		appKeepers.keys[feemarkettypes.StoreKey],
		appKeepers.tkeys[feemarkettypes.TransientKey],
	)

	appKeepers.EvmKeeper = evmkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[evmtypes.StoreKey],
		appKeepers.tkeys[evmtypes.TransientKey],
		appKeepers.GetSubspace(evmtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		&stakingKeeper,
		appKeepers.FeeMarketKeeper,
		evmTrace,
	)

	appKeepers.Erc20Keeper = erc20keeper.NewKeeper(
		appKeepers.keys[erc20types.StoreKey], appCodec, appKeepers.GetSubspace(erc20types.ModuleName),
		appKeepers.AccountKeeper, appKeepers.BankKeeper, appKeepers.EvmKeeper,
	)

	appKeepers.EvmKeeper = appKeepers.EvmKeeper.SetHooks(
		evmkeeper.NewMultiEvmHooks(
			appKeepers.Erc20Keeper.Hooks(),
		),
	)

	// RouterKeeper must be created before TransferKeeper
	appKeepers.RouterKeeper = routerkeeper.NewKeeper(
		appCodec, appKeepers.keys[routertypes.StoreKey],
		appKeepers.GetSubspace(routertypes.ModuleName),
		appKeepers.TransferKeeper, // Will be zero-value here. Reference is set later on with SetTransferKeeper.
		appKeepers.IBCKeeper.ChannelKeeper,
		appKeepers.DistrKeeper,
		appKeepers.BankKeeper,
		// The ICS4Wrapper is replaced by the IBCFeeKeeper instead of the channel so that sending can be overridden by the middleware
		&appKeepers.IBCFeeKeeper,
	)

	appKeepers.IBCFeeKeeper = ibcfeekeeper.NewKeeper(
		appCodec, appKeepers.keys[ibcfeetypes.StoreKey],
		appKeepers.GetSubspace(ibcfeetypes.ModuleName), // this isn't even used in the keeper but is required?
		appKeepers.IBCKeeper.ChannelKeeper,             // replaced with IBC middleware
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
	)

	appKeepers.TransferKeeper = ibctransferkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[ibctransfertypes.StoreKey],
		appKeepers.GetSubspace(ibctransfertypes.ModuleName),
		// The ICS4Wrapper is replaced by the RouterKeeper instead of the channel so that sending can be overridden by the middleware
		appKeepers.RouterKeeper,
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.ScopedTransferKeeper,
	)

	appKeepers.RouterKeeper.SetTransferKeeper(appKeepers.TransferKeeper)

	appKeepers.ICAHostKeeper = icahostkeeper.NewKeeper(
		appCodec, appKeepers.keys[icahosttypes.StoreKey],
		appKeepers.GetSubspace(icahosttypes.SubModuleName),
		appKeepers.IBCKeeper.ChannelKeeper,
		&appKeepers.IBCKeeper.PortKeeper,
		appKeepers.AccountKeeper,
		appKeepers.ScopedICAHostKeeper,
		bApp.MsgServiceRouter(),
	)

	// ICA Controller keeper
	appKeepers.ICAControllerKeeper = icacontrollerkeeper.NewKeeper(
		appCodec, appKeepers.keys[icacontrollertypes.StoreKey], appKeepers.GetSubspace(icacontrollertypes.SubModuleName),
		appKeepers.IBCFeeKeeper, // use ics29 fee as ics4Wrapper in middleware stack
		appKeepers.IBCKeeper.ChannelKeeper, &appKeepers.IBCKeeper.PortKeeper,
		appKeepers.ScopedICAControllerKeeper, bApp.MsgServiceRouter(),
	)

	icaHostIBCModule := icahost.NewIBCModule(appKeepers.ICAHostKeeper)

	// initialize ICA module with mock module as the authentication module on the controller side
	var icaControllerStack porttypes.IBCModule
	icaControllerStack = icacontroller.NewIBCMiddleware(icaControllerStack, appKeepers.ICAControllerKeeper)
	icaControllerStack = ibcfee.NewIBCMiddleware(icaControllerStack, appKeepers.IBCFeeKeeper)

	// Create Transfer Stack
	var transferStack porttypes.IBCModule
	transferStack = transfer.NewIBCModule(appKeepers.TransferKeeper)
	transferStack = router.NewIBCMiddleware(
		transferStack,
		appKeepers.RouterKeeper,
		0,
		routerkeeper.DefaultForwardTransferPacketTimeoutTimestamp,
		routerkeeper.DefaultRefundTransferPacketTimeoutTimestamp,
	)

	transferStack = ibcfee.NewIBCMiddleware(transferStack, appKeepers.IBCFeeKeeper)

	// RecvPacket, message that originates from core IBC and goes down to app, the flow is:
	// channel.RecvPacket -> fee.OnRecvPacket -> icaHost.OnRecvPacket
	icaHostStack := ibcfee.NewIBCMiddleware(icaHostIBCModule, appKeepers.IBCFeeKeeper)

	var wasmStack porttypes.IBCModule
	wasmStack = wasm.NewIBCHandler(appKeepers.WasmKeeper, appKeepers.IBCKeeper.ChannelKeeper, &appKeepers.IBCFeeKeeper)
	wasmStack = ibcfee.NewIBCMiddleware(wasmStack, appKeepers.IBCFeeKeeper)

	// create static IBC router, add transfer route, then set and seal it
	ibcRouter := porttypes.NewRouter()
	ibcRouter.AddRoute(icahosttypes.SubModuleName, icaHostStack).
		AddRoute(wasm.ModuleName, wasmStack).
		AddRoute(ibctransfertypes.ModuleName, transferStack).
		AddRoute(icacontrollertypes.SubModuleName, icaControllerStack)

	appKeepers.IBCKeeper.SetRouter(ibcRouter)

	// create evidence keeper with router
	appKeepers.EvidenceKeeper = *evidencekeeper.NewKeeper(
		appCodec,
		appKeepers.keys[evidencetypes.StoreKey],
		&appKeepers.StakingKeeper,
		appKeepers.SlashingKeeper,
	)

	appKeepers.RewardKeeper = rewardkeeper.NewKeeper(
		appCodec,
		appKeepers.keys[rewardtypes.StoreKey],
		appKeepers.GetSubspace(rewardtypes.ModuleName),
		appKeepers.AccountKeeper,
		appKeepers.BankKeeper,
		appKeepers.StakingKeeper,
		appKeepers.DistrKeeper,
		appKeepers.MintKeeper,
	)

	return appKeepers
}

// GetSubspace returns a param subspace for a given module name.
func (appKeepers *AppKeepers) GetSubspace(moduleName string) paramstypes.Subspace {
	subspace, _ := appKeepers.ParamsKeeper.GetSubspace(moduleName)
	return subspace
}

// initParamsKeeper init params keeper and its subspaces
func initParamsKeeper(appCodec codec.BinaryCodec, legacyAmino *codec.LegacyAmino, key, tkey sdk.StoreKey) paramskeeper.Keeper {
	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, key, tkey)

	paramsKeeper.Subspace(authtypes.ModuleName)
	paramsKeeper.Subspace(banktypes.ModuleName)
	paramsKeeper.Subspace(stakingtypes.ModuleName)
	paramsKeeper.Subspace(minttypes.ModuleName)
	paramsKeeper.Subspace(distrtypes.ModuleName)
	paramsKeeper.Subspace(slashingtypes.ModuleName)
	paramsKeeper.Subspace(govtypes.ModuleName).WithKeyTable(govtypes.ParamKeyTable())
	paramsKeeper.Subspace(crisistypes.ModuleName)
	paramsKeeper.Subspace(ibctransfertypes.ModuleName)
	paramsKeeper.Subspace(ibchost.ModuleName)
	paramsKeeper.Subspace(routertypes.ModuleName).WithKeyTable(routertypes.ParamKeyTable())
	paramsKeeper.Subspace(icacontrollertypes.SubModuleName)
	paramsKeeper.Subspace(icahosttypes.SubModuleName)
	paramsKeeper.Subspace(wasm.ModuleName)
	paramsKeeper.Subspace(feemarkettypes.ModuleName)
	paramsKeeper.Subspace(evmtypes.ModuleName)
	paramsKeeper.Subspace(erc20types.ModuleName)
	paramsKeeper.Subspace(rewardtypes.ModuleName).WithKeyTable(rewardtypes.ParamKeyTable())
	paramsKeeper.Subspace(volunteertypes.ModuleName)

	return paramsKeeper
}

// GetIBCKeeper implements the TestingApp interface.
func (appKeepers *AppKeepers) GetIBCKeeper() *ibckeeper.Keeper {
	return appKeepers.IBCKeeper
}

// GetScopedIBCKeeper implements the TestingApp interface.
func (appKeepers *AppKeepers) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return appKeepers.ScopedIBCKeeper
}
