package app

import (
	"testing"

	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/server"
	"github.com/stretchr/testify/require"
)

type mempoolTestAppOptions map[string]interface{}

func (opts mempoolTestAppOptions) Get(key string) interface{} {
	return opts[key]
}

func TestConfigureEVMMempoolRejectsDisabledAppMempool(t *testing.T) {
	xpla := newTestApp(t)

	err := xpla.configureEVMMempool(
		mempoolTestAppOptions{server.FlagMempoolMaxTxs: -1},
		log.NewNopLogger(),
	)
	require.ErrorContains(t, err, "mempool.max-txs=-1 is unsupported")
}
