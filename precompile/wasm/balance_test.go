package wasm

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	pbank "github.com/xpladev/xpla/precompile/bank"
)

func TestBalanceABI(t *testing.T) {
	method, ok := ABI.Methods["balance"]
	require.True(t, ok, "IWasm must expose balance(address,string)")
	require.Equal(t, "balance(address,string)", method.Sig)
	require.Equal(t, "view", method.StateMutability)
	require.Equal(t, "16cadeab", fmt.Sprintf("%x", method.ID))
	require.Len(t, method.Outputs, 1)
	require.Equal(t, "uint256", method.Outputs[0].Type.String())
	require.False(t, (PrecompiledWasm{}).IsTransaction(&method))
	require.Len(t, method.Inputs, 2)
	require.Equal(t, "address", method.Inputs[0].Type.String())
	require.Equal(t, "string", method.Inputs[1].Type.String())
	for name, other := range ABI.Methods {
		if name != "balance" {
			require.NotEqual(t, method.ID, other.ID, "selector collision with %s", name)
		}
	}

}

type balanceBankKeeper struct {
	pbank.BankKeeper
	calls   int
	address sdk.AccAddress
	denom   string
	amount  int64
}

func (k *balanceBankKeeper) GetBalance(_ context.Context, address sdk.AccAddress, denom string) sdk.Coin {
	k.calls++
	k.address, k.denom = address, denom
	return sdk.NewInt64Coin(denom, k.amount)
}

func TestBalanceDenoms(t *testing.T) {
	address := common.HexToAddress("0x1234")
	fullAddress := sdk.AccAddress(append(bytes.Repeat([]byte{1}, 12), address.Bytes()...))
	for _, tc := range []struct {
		name, denom string
		amount      int64
		wantError   bool
	}{
		{"native", "axpla", 42, false},
		{"IBC", "ibc/" + strings.Repeat("A", 64), 91, false},
		{"unheld", "uatom", 0, false},
		{"invalid", "!", 0, true},
		{"empty", "", 0, true},
		{"ERC20", "xerc20:" + address.Hex(), 0, true},
		{"CW20", "xcw20:" + fullAddress.String(), 0, true},
		{"malformed ERC20", "xerc20:invalid", 0, true},
		{"malformed CW20", "xcw20:invalid", 0, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bank := &balanceBankKeeper{amount: tc.amount}
			p := NewPrecompiledWasm(nil, &stubWasmKeeper{resolvedAddress: fullAddress}, bank)
			method := ABI.Methods["balance"]
			result, err := p.balance(sdk.Context{}, &method, []interface{}{address, tc.denom})
			if tc.wantError {
				require.Error(t, err)
				require.Zero(t, bank.calls)
				return
			}
			require.NoError(t, err)
			values, err := method.Outputs.Unpack(result)
			require.NoError(t, err)
			require.Zero(t, big.NewInt(tc.amount).Cmp(values[0].(*big.Int)))
			require.Equal(t, fullAddress, bank.address)
			require.Equal(t, tc.denom, bank.denom)
			require.Equal(t, 1, bank.calls)
		})
	}
}

func TestBalanceContractResolution(t *testing.T) {
	address := common.HexToAddress("0x1234")
	resolverErr := errors.New("alias lookup failed")
	for _, tc := range []struct {
		name      string
		keeper    stubWasmKeeper
		wantError string
	}{
		{"exact twenty byte contract", stubWasmKeeper{resolvedAddress: sdk.AccAddress(address.Bytes())}, ""},
		{"missing contract", stubWasmKeeper{resolveErr: wasmtypes.ErrNoSuchContractFn(sdk.AccAddress(address.Bytes()).String())}, "no such contract"},
		{"resolver error", stubWasmKeeper{resolveErr: resolverErr}, resolverErr.Error()},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bank := &balanceBankKeeper{amount: 7}
			p := NewPrecompiledWasm(nil, &tc.keeper, bank)
			method := ABI.Methods["balance"]
			_, err := p.balance(sdk.Context{}, &method, []interface{}{address, "axpla"})
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.Zero(t, bank.calls)
				return
			}
			require.NoError(t, err)
			require.Equal(t, sdk.AccAddress(address.Bytes()), bank.address)
			require.Equal(t, 1, bank.calls)
		})
	}
}
