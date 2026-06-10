package cmd

import (
	"testing"

	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/stretchr/testify/require"

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
}
