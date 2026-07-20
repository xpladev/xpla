package app

import (
	"testing"

	"github.com/stretchr/testify/require"

	cosmosevmutils "github.com/cosmos/evm/utils"

	xplaprecompile "github.com/xpladev/xpla/precompile"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
)

func TestBlockedModuleAccountAddrsIncludesXplaStaticPrecompiles(t *testing.T) {
	xpla := newTestApp(t)

	blockedAddrs := xpla.BlockedModuleAccountAddrs(xpla.ModuleAccountAddrs())
	for _, precompile := range xplaprecompile.PrecompiledAddressesXpla {
		bech32Addr := cosmosevmutils.Bech32StringFromHexAddress(precompile.Hex())
		require.Truef(t, blockedAddrs[bech32Addr], "expected %s to be blocked", precompile.Hex())
	}
	require.Contains(t, xplaprecompile.PrecompiledAddressesXpla, pwasm.DelegatecallAddress)
}
