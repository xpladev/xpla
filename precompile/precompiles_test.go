package precompile

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	pwasm "github.com/xpladev/xpla/precompile/wasm"
	xplabankkeeper "github.com/xpladev/xpla/x/bank/keeper"
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

func TestNewAvailableStaticPrecompilesWrapsICS20StateDBContext(t *testing.T) {
	precompiles := NewAvailableStaticPrecompiles(
		stakingkeeper.Keeper{},
		distributionkeeper.Keeper{},
		nil,
		nil,
		nil,
		&evmkeeper.Keeper{},
		govkeeper.Keeper{},
		slashingkeeper.Keeper{},
		nil,
		xplabankkeeper.Keeper{},
		nil,
		nil,
		nil,
		nil,
	)

	ics20, ok := precompiles[common.HexToAddress(evmtypes.ICS20PrecompileAddress)]
	require.True(t, ok)
	require.IsType(t, ics20Precompile{}, ics20)
	require.NotNil(t, ics20.(ics20Precompile).Precompile)

	wasmDelegate, ok := precompiles[pwasm.DelegatecallAddress]
	require.True(t, ok)
	require.IsType(t, wasmDelegatePrecompile{}, wasmDelegate)
}
