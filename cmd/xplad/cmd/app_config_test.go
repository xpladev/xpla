package cmd

import (
	"os"
	"path/filepath"
	"testing"

	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	evmserver "github.com/cosmos/evm/server"
	cosmosevmserverconfig "github.com/cosmos/evm/server/config"
	ibctransfertypes "github.com/cosmos/ibc-go/v11/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v11/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v11/modules/core/04-channel/types"

	xplaparams "github.com/xpladev/xpla/app/params"
)

func TestInitAppConfigUsesRequiredEVMMempool(t *testing.T) {
	appTemplate, appConfig := initAppConfig()
	customAppConfig, ok := appConfig.(xplaparams.CustomAppConfig)
	require.True(t, ok)

	require.Equal(t, 0, customAppConfig.Mempool.MaxTxs)
	require.Contains(t, appTemplate, "Setting max-txs to negative 1 (-1) is not supported.")
	require.NotContains(t, appTemplate, "will disable transactions from being inserted into the mempool")
	require.Equal(t, tmcfg.MempoolTypeApp, initCometConfig().Mempool.Type)
	require.Equal(t, []string{
		sdk.MsgTypeURL(&ibcchanneltypes.MsgRecvPacket{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgAcknowledgement{}),
		sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeout{}),
		sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeoutOnClose{}),
	}, customAppConfig.BypassMinFeeMsgTypes)
	require.Contains(t, appTemplate, sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}))
	require.Contains(t, appTemplate, sdk.MsgTypeURL(&ibcchanneltypes.MsgTimeout{}))
}

func TestInitAppConfigRejectsLegacyCometMempool(t *testing.T) {
	_, appConfig := initAppConfig()
	customAppConfig, ok := appConfig.(xplaparams.CustomAppConfig)
	require.True(t, ok)

	legacyCometConfig := initCometConfig()
	legacyCometConfig.Mempool.Type = tmcfg.MempoolTypeFlood

	err := cosmosevmserverconfig.ValidateCrossConfig(legacyCometConfig, &customAppConfig.Config)
	require.ErrorContains(t, err, "config.toml:mempool.type")
	require.ErrorContains(t, err, "want 'app', got 'flood'")
}

func TestAddModuleInitFlagsAllowsAppTomlMempoolMaxTxs(t *testing.T) {
	const appTomlMaxTxs = 5000

	home := t.TempDir()
	configDir := filepath.Join(home, "config")
	require.NoError(t, os.Mkdir(configDir, 0o700))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, "app.toml"),
		[]byte("[mempool]\nmax-txs = 5000\n"),
		0o600,
	))

	startCmd := evmserver.StartCmd(evmserver.NewDefaultStartOptions(nil, home))
	mempoolMaxTxsFlag := startCmd.Flags().Lookup(server.FlagMempoolMaxTxs)
	require.NotNil(t, mempoolMaxTxsFlag)
	require.True(t, mempoolMaxTxsFlag.Changed)

	addModuleInitFlags(startCmd)
	require.False(t, mempoolMaxTxsFlag.Changed)
	require.NoError(t, startCmd.Flags().Set(flags.FlagHome, home))

	appTemplate, appConfig := initAppConfig()
	serverCtx, err := server.InterceptConfigsAndCreateContext(
		startCmd,
		appTemplate,
		appConfig,
		initCometConfig(),
	)
	require.NoError(t, err)
	require.Equal(t, appTomlMaxTxs, serverCtx.Viper.GetInt(server.FlagMempoolMaxTxs))
	require.Equal(t, "5000", mempoolMaxTxsFlag.Value.String())
}
