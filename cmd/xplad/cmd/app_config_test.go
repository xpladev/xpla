package cmd

import (
	"testing"

	tmcfg "github.com/cometbft/cometbft/config"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
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
