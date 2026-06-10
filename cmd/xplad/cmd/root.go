package cmd

import (
	"errors"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tmcfg "github.com/cometbft/cometbft/config"
	tmcli "github.com/cometbft/cometbft/libs/cli"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/client/v2/autocli"
	"cosmossdk.io/log/v2"
	sdkmath "cosmossdk.io/math"
	confixcmd "cosmossdk.io/tools/confix/cmd"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/config"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/pruning"
	"github.com/cosmos/cosmos-sdk/client/rpc"
	"github.com/cosmos/cosmos-sdk/client/snapshot"
	"github.com/cosmos/cosmos-sdk/crypto/ledger"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	serverconfig "github.com/cosmos/cosmos-sdk/server/config"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authcmd "github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtxconfig "github.com/cosmos/cosmos-sdk/x/auth/tx/config"
	"github.com/cosmos/cosmos-sdk/x/auth/types"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	ibctransfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"

	"github.com/CosmWasm/wasmd/x/wasm"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	evmclient "github.com/cosmos/evm/client"
	"github.com/cosmos/evm/client/debug"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/evm/crypto/hd"
	evmaddress "github.com/cosmos/evm/encoding/address"
	evmserver "github.com/cosmos/evm/server"
	evmcfg "github.com/cosmos/evm/server/config"

	xpla "github.com/xpladev/xpla/app"
	"github.com/xpladev/xpla/app/encoding"
	"github.com/xpladev/xpla/app/params"
	legacykeyclient "github.com/xpladev/xpla/legacy/ethermint/client"
	xplatypes "github.com/xpladev/xpla/types"
)

// NewRootCmd creates a new root command for simd. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	// we "pre"-instantiate the application for getting the injected/configured encoding configuration
	initAppOptions := viper.New()
	tempDir := tempDir()
	initAppOptions.Set(flags.FlagHome, tempDir)
	tempApplication := xpla.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		true,
		map[int64]bool{},
		tempDir,
		initAppOptions,
		xpla.EmptyWasmOptions,
	)
	defer func() {
		if err := tempApplication.Close(); err != nil {
			panic(err)
		}
	}()

	initClientCtx := client.Context{}.
		WithCodec(tempApplication.AppCodec()).
		WithInterfaceRegistry(tempApplication.InterfaceRegistry()).
		WithLegacyAmino(tempApplication.LegacyAmino()).
		WithInput(os.Stdin).
		WithAccountRetriever(types.AccountRetriever{}).
		WithHomeDir(xpla.DefaultNodeHome).
		WithKeyringOptions(hd.EthSecp256k1Option()).
		WithViper("XPLA")

	rootCmd := &cobra.Command{
		Use:   "xplad",
		Short: "xpla App",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			cmd.SetOut(cmd.OutOrStdout())
			cmd.SetErr(cmd.ErrOrStderr())

			initClientCtx = initClientCtx.WithCmdContext(cmd.Context())
			initClientCtx, err := client.ReadPersistentCommandFlags(initClientCtx, cmd.Flags())
			if err != nil {
				return err
			}

			initClientCtx, err = config.ReadFromClientConfig(initClientCtx)
			if err != nil {
				return err
			}

			// This needs to go after ReadFromClientConfig, as that function
			// sets the RPC client needed for SIGN_MODE_TEXTUAL. This sign mode
			// is only available if the client is online.
			if !initClientCtx.Offline {
				txConfig := encoding.NewTxConfig(initClientCtx.Codec, tx.DefaultSignModes)
				txConfigOpts := tx.ConfigOptions{
					EnabledSignModes:           append(tx.DefaultSignModes, signing.SignMode_SIGN_MODE_TEXTUAL),
					TextualCoinMetadataQueryFn: authtxconfig.NewGRPCCoinMetadataQueryFn(initClientCtx),
				}
				txConfig.TxConfig, err = tx.NewTxConfigWithOptions(
					initClientCtx.Codec,
					txConfigOpts,
				)
				if err != nil {
					return err
				}
				initClientCtx = initClientCtx.WithTxConfig(txConfig)
			}

			if err = client.SetCmdClientContextHandler(initClientCtx, cmd); err != nil {
				return err
			}

			xplaTemplate, xplaAppConfig := initAppConfig()
			customCometConfig := initCometConfig()

			return server.InterceptConfigsPreRunHandler(cmd, xplaTemplate, xplaAppConfig, customCometConfig)
		},
	}

	initRootCmd(rootCmd, tempApplication, tempApplication.GetTxConfig())

	autoCliOpts := enrichAutoCliOpts(tempApplication.AutoCliOpts(), initClientCtx)
	if err := autoCliOpts.EnhanceRootCommand(rootCmd); err != nil {
		panic(err)
	}

	ledger.SetCreatePubkey(func(key []byte) cryptotypes.PubKey {
		return &ethsecp256k1.PubKey{Key: key}
	})

	return rootCmd
}

func enrichAutoCliOpts(autoCliOpts autocli.AppOptions, clientCtx client.Context) autocli.AppOptions {
	autoCliOpts.AddressCodec = evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix())
	autoCliOpts.ValidatorAddressCodec = evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix())
	autoCliOpts.ConsensusAddressCodec = evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix())

	autoCliOpts.ClientCtx = clientCtx

	return autoCliOpts
}

// initCometConfig helps to override default CometBFT Config values.
// return tmcfg.DefaultConfig if no custom configuration is required for the application.
func initCometConfig() *tmcfg.Config {
	cfg := tmcfg.DefaultConfig()

	// XPLA uses Cosmos EVM's app-side mempool as the default and required tx path.
	cfg.Mempool.Type = tmcfg.MempoolTypeApp

	// these values put a higher strain on node memory
	// cfg.P2P.MaxNumInboundPeers = 100
	// cfg.P2P.MaxNumOutboundPeers = 40

	return cfg
}

func initAppConfig() (string, interface{}) {
	customAppConfig := evmcfg.Config{
		Config:  *serverconfig.DefaultConfig(),
		EVM:     *evmcfg.DefaultEVMConfig(),
		JSONRPC: *evmcfg.DefaultJSONRPCConfig(),
		TLS:     *evmcfg.DefaultTLSConfig(),
	}
	customAppTemplate := serverconfig.DefaultConfigTemplate + evmcfg.DefaultEVMConfigTemplate

	customAppConfig.StateSync.SnapshotInterval = 1000
	customAppConfig.StateSync.SnapshotKeepRecent = 10
	customAppConfig.Mempool.MaxTxs = 0
	customAppConfig.EVM.EVMChainID = 37

	return params.CustomConfigTemplate(customAppTemplate), params.CustomAppConfig{
		Config: customAppConfig,
		BypassMinFeeMsgTypes: []string{
			sdk.MsgTypeURL(&ibcchanneltypes.MsgRecvPacket{}),
			sdk.MsgTypeURL(&ibcchanneltypes.MsgAcknowledgement{}),
			sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
			sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}),
			sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeout{}),
			sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeoutOnClose{}),
		},
	}
}

func initRootCmd(rootCmd *cobra.Command,
	xplaApp *xpla.XplaApp,
	txConfig client.TxConfig,
) {
	cfg := sdk.GetConfig()
	cfg.Seal()
	basicManager := xplaApp.ModuleBasics

	ac := appCreator{}
	sdkAppCreatorWrapper := func(l log.Logger, d dbm.DB, ao servertypes.AppOptions) servertypes.Application {
		return ac.newApp(l, d, ao)
	}

	rootCmd.AddCommand(
		InitCmd(xplaApp, xpla.DefaultNodeHome),
		tmcli.NewCompletionCmd(rootCmd, true),
		debug.Cmd(),
		confixcmd.ConfigCommand(),
		pruning.Cmd(sdkAppCreatorWrapper, xpla.DefaultNodeHome),
		snapshot.Cmd(sdkAppCreatorWrapper),
	)

	evmserver.AddCommands(rootCmd, evmserver.NewDefaultStartOptions(ac.newApp, xpla.DefaultNodeHome), ac.appExport, addModuleInitFlags)

	// add keybase, auxiliary RPC, query, and tx child commands
	keysCmd := evmclient.KeyCommands(xpla.DefaultNodeHome, true)
	keysCmd.AddCommand(legacykeyclient.UnsafeExportLegacyEthKeyCommand())
	rootCmd.AddCommand(
		server.StatusCommand(),
		genesisCommand(txConfig, basicManager, AddGenesisAccountCmd(xpla.DefaultNodeHome)),
		queryCommand(),
		txCommand(basicManager),
		keysCmd,
	)
}

func addModuleInitFlags(startCmd *cobra.Command) {
	wasm.AddModuleInitFlags(startCmd)

	// min-gas-price follows evm/feemarket module
	startCmd.Flags().Set(server.FlagMinGasPrices, sdkmath.ZeroInt().String()+xplatypes.DefaultDenom)
}

// genesisCommand builds genesis-related `simd genesis` command. Users may provide application specific commands as a parameter
func genesisCommand(txConfig client.TxConfig, basicManager module.BasicManager, cmds ...*cobra.Command) *cobra.Command {
	cmd := genutilcli.GenesisCoreCommand(txConfig, basicManager, xpla.DefaultNodeHome)

	// remove default add-genesis-account commands
	for _, subCmd := range cmd.Commands() {
		if subCmd.Name() == "add-genesis-account" || subCmd.Name() == "bulk-add-genesis-account" {
			cmd.RemoveCommand(subCmd)
		}
	}

	// change add-genesis-account to custom AddGenesisAccountCmd()
	for _, subCmd := range cmds {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func queryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "query",
		Aliases:                    []string{"q"},
		Short:                      "Querying subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		rpc.ValidatorCommand(),
		server.QueryBlocksCmd(),
		server.QueryBlockCmd(),
		server.QueryBlockResultsCmd(),
		authcmd.QueryTxsByEventsCmd(),
		authcmd.QueryTxCmd(),
	)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

func txCommand(basicManager module.BasicManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "tx",
		Short:                      "Transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(
		authcmd.GetSignCommand(),
		authcmd.GetSignBatchCommand(),
		authcmd.GetMultiSignCommand(),
		authcmd.GetMultiSignBatchCmd(),
		authcmd.GetValidateSignaturesCommand(),
		flags.LineBreak,
		authcmd.GetBroadcastCommand(),
		authcmd.GetEncodeCommand(),
		authcmd.GetDecodeCommand(),
	)

	// NOTE: this must be registered for now so that submit-legacy-proposal
	// message (e.g. consumer-addition proposal) can be routed to the its handler and processed correctly.
	basicManager.AddTxCommands(cmd)

	cmd.PersistentFlags().String(flags.FlagChainID, "", "The network chain ID")

	return cmd
}

type appCreator struct{}

func (a appCreator) newApp(
	logger log.Logger,
	db dbm.DB,
	appOpts servertypes.AppOptions,
) evmserver.Application {
	var wasmOpts []wasmkeeper.Option
	if cast.ToBool(appOpts.Get("telemetry.enabled")) {
		wasmOpts = append(wasmOpts, wasmkeeper.WithVMCacheMetrics(prometheus.DefaultRegisterer))
	}

	skipUpgradeHeights := make(map[int64]bool)
	for _, h := range cast.ToIntSlice(appOpts.Get(server.FlagUnsafeSkipUpgrades)) {
		skipUpgradeHeights[int64(h)] = true
	}

	home := cast.ToString(appOpts.Get(flags.FlagHome))

	return xpla.NewXplaApp(
		logger,
		db,
		true,
		skipUpgradeHeights,
		home,
		appOpts,
		wasmOpts,
		server.DefaultBaseappOptions(appOpts)...,
	)
}

func (a appCreator) appExport(
	logger log.Logger,
	db dbm.DB,
	height int64,
	forZeroHeight bool,
	jailAllowedAddrs []string,
	appOpts servertypes.AppOptions,
	modulesToExport []string,
) (servertypes.ExportedApp, error) {

	homePath, ok := appOpts.Get(flags.FlagHome).(string)
	if !ok || homePath == "" {
		return servertypes.ExportedApp{}, errors.New("application home is not set")
	}

	// InvCheckPeriod
	viperAppOpts, ok := appOpts.(*viper.Viper)
	if !ok {
		return servertypes.ExportedApp{}, errors.New("appOpts is not viper.Viper")
	}
	// overwrite the FlagInvCheckPeriod
	viperAppOpts.Set(server.FlagInvCheckPeriod, 1)
	appOpts = viperAppOpts

	var loadLatest bool
	if height == -1 {
		loadLatest = true
	}

	var emptyWasmOpts []wasmkeeper.Option
	xplaApp := xpla.NewXplaApp(
		logger,
		db,
		loadLatest,
		map[int64]bool{},
		homePath,
		appOpts,
		emptyWasmOpts,
	)

	if height != -1 {
		if err := xplaApp.LoadHeight(height); err != nil {
			return servertypes.ExportedApp{}, err
		}
	}

	return xplaApp.ExportAppStateAndValidators(forZeroHeight, jailAllowedAddrs, modulesToExport)
}

var tempDir = func() string {
	dir, err := os.MkdirTemp("", ".xpla")
	if err != nil {
		dir = xpla.DefaultNodeHome
	}
	defer os.RemoveAll(dir)

	return dir
}
