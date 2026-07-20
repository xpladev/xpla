package ante

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtensionOptionTypeURLs(t *testing.T) {
	require.Equal(t, "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx", extensionOptionsEthereumTxTypeURL)
	require.Equal(t, "/cosmos.evm.ante.v1.ExtensionOptionDynamicFeeTx", extensionOptionDynamicFeeTxTypeURL)
}
