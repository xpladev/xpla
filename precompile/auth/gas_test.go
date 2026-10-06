package auth_test

import (
	"bytes"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/statedb"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	pauth "github.com/xpladev/xpla/precompile/auth"
	"github.com/xpladev/xpla/tests/integration/testutil"
)

func TestAccountEntryPointGasSettlement(t *testing.T) {
	alias := sdk.AccAddress(bytes.Repeat([]byte{0x31}, 20))
	target := sdk.AccAddress(append(bytes.Repeat([]byte{0x32}, 12), alias...))
	tests := []struct {
		name             string
		accountAddresses []sdk.AccAddress
		registerAlias    bool
		expected         string
	}{
		{
			name:             "alias",
			accountAddresses: []sdk.AccAddress{target},
			registerAlias:    true,
			expected:         target.String(),
		},
		{
			name:             "exact",
			accountAddresses: []sdk.AccAddress{target, alias},
			registerAlias:    true,
			expected:         alias.String(),
		},
		{
			name:     "missing",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := testutil.CreateTestInput(t)
			for _, address := range tt.accountAddresses {
				account := input.AccountKeeper.NewAccountWithAddress(input.Ctx, address)
				input.AccountKeeper.SetAccount(input.Ctx, account)
			}
			if tt.registerAlias {
				require.NoError(t, input.App.WasmKeeper.RegisterWasmAlias(input.Ctx, alias, target))
			}

			params := input.App.EvmKeeper.GetParams(input.Ctx)
			params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, pauth.Address.String())
			registered, found, err := input.App.EvmKeeper.GetStaticPrecompileInstance(&params, pauth.Address)
			require.NoError(t, err)
			require.True(t, found)
			p, ok := registered.(pauth.PrecompiledAuth)
			require.True(t, ok)
			data, err := pauth.ABI.Pack("account", common.BytesToAddress(alias))
			require.NoError(t, err)
			const budget, initialGas = uint64(1_000_000), uint64(30)
			newContext := func() sdk.Context {
				ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).
					WithKVGasConfig(p.KvGasConfig).WithTransientKVGasConfig(p.TransientKVGasConfig)
				ctx.GasMeter().ConsumeGas(initialGas, "before auth query")
				return ctx
			}
			newContract := func() *vm.Contract {
				contract := vm.NewContract(common.Address{}, pauth.Address, uint256.NewInt(0), budget, nil)
				contract.Input = data
				return contract
			}

			// Measure the native query on identical state, then verify the registered
			// Run entry point charges that SDK delta to EVM gas exactly once.
			measured := newContext()
			nativeOutput, err := p.Execute(measured, nil, newContract(), true)
			require.NoError(t, err)
			sdkGas := measured.GasMeter().GasConsumed() - initialGas
			require.Positive(t, sdkGas)
			db := statedb.New(newContext(), input.App.EvmKeeper, statedb.NewEmptyTxConfig())
			evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
			contract := newContract()
			output, err := p.Run(evm, contract, true)
			require.NoError(t, err)
			require.Equal(t, nativeOutput, output)
			require.Equal(t, sdkGas, budget-contract.Gas)
			values, err := pauth.ABI.Unpack("account", output)
			require.NoError(t, err)
			require.Equal(t, []interface{}{tt.expected}, values)
			t.Logf("native SDK gas / Run EVM gas: %d", sdkGas)
		})
	}
}
