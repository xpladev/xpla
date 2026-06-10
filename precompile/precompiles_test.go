package precompile

import (
	"testing"

	"github.com/stretchr/testify/require"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestValidateActiveStaticPrecompiles(t *testing.T) {
	require.NoError(t, ValidateActiveStaticPrecompiles(DefaultActiveStaticPrecompiles()))

	for _, address := range []string{
		evmtypes.VestingPrecompileAddress,
		evmtypes.BankPrecompileAddress,
		"0x0000000000000000000000000000000000009999",
	} {
		t.Run(address, func(t *testing.T) {
			require.ErrorContains(t, ValidateActiveStaticPrecompiles([]string{address}), "unsupported active static precompile")
		})
	}
}
