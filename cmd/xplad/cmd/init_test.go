package cmd

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestInitPrintInfoMatchesSDKJSONSchema(t *testing.T) {
	bz, err := json.Marshal(initPrintInfo{})
	require.NoError(t, err)
	require.JSONEq(t, `{
		"moniker": "",
		"chain_id": "",
		"node_id": "",
		"gentxs_dir": "",
		"app_message": null
	}`, string(bz))
}
