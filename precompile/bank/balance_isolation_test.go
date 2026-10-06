package bank_test

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	"github.com/cosmos/evm/x/vm/statedb"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	pbank "github.com/xpladev/xpla/precompile/bank"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

// nestedWasmBankKeeper models the native bank submessages emitted while the
// bank precompile is executing a CW20 contract. The outer precompile's balance
// handler must not project the 32-byte contract events into its 20-byte suffix.
type nestedWasmBankKeeper struct {
	pbank.BankKeeper

	sender    sdk.AccAddress
	contract  sdk.AccAddress
	recipient sdk.AccAddress
	denom     string
}

func (k nestedWasmBankKeeper) SendCoins(ctx context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	if err := k.BankKeeper.SendCoins(ctx, k.sender, k.contract, sdk.NewCoins(sdk.NewInt64Coin(k.denom, 7))); err != nil {
		return err
	}
	return k.BankKeeper.SendCoins(ctx, k.contract, k.recipient, sdk.NewCoins(sdk.NewInt64Coin(k.denom, 3)))
}

func TestBankPrecompileNestedWasmEventsPreserveExactBankBalanceOwnership(t *testing.T) {
	input := testutil.CreateTestInput(t)
	sender := common.HexToAddress("0x1111111111111111111111111111111111111111")
	alias := common.HexToAddress("0x2222222222222222222222222222222222222222")
	outerRecipient := common.HexToAddress("0x3333333333333333333333333333333333333333")
	recipient := common.HexToAddress("0x4444444444444444444444444444444444444444")

	senderAddr := sdk.AccAddress(sender.Bytes())
	aliasAddr := sdk.AccAddress(alias.Bytes())
	contractAddr := sdk.AccAddress(append(bytes.Repeat([]byte{0x55}, 12), alias.Bytes()...))
	recipientAddr := sdk.AccAddress(recipient.Bytes())
	coins := func(amount int64) sdk.Coins {
		return sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(amount)))
	}
	for _, account := range []struct {
		address sdk.AccAddress
		amount  int64
	}{{senderAddr, 1000}, {aliasAddr, 10}, {contractAddr, 30}, {recipientAddr, 5}} {
		require.NoError(t, input.InitAccountWithCoins(account.address, coins(account.amount)))
	}

	keeper := nestedWasmBankKeeper{
		BankKeeper: input.BankKeeper,
		sender:     senderAddr,
		contract:   contractAddr,
		recipient:  recipientAddr,
		denom:      xplatypes.DefaultDenom,
	}
	precompile := pbank.NewPrecompiledBank(keeper)

	const budget = uint64(1_000_000)
	ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).WithEventManager(sdk.NewEventManager())
	db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
	require.Equal(t, uint64(1000), db.GetBalance(sender).Uint64())
	require.Equal(t, uint64(10), db.GetBalance(alias).Uint64())
	require.Equal(t, uint64(5), db.GetBalance(recipient).Uint64())
	require.False(t, input.BankKeeper.BlockedAddr(contractAddr))

	evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
	evm.Origin = sender
	contract := vm.NewContract(sender, pbank.Address, uint256.NewInt(0), budget, nil)
	var err error
	contract.Input, err = pbank.ABI.Pack("send", sender, outerRecipient, cmn.NewCoinsResponse(coins(1)))
	require.NoError(t, err)

	_, err = precompile.Run(evm, contract, false)
	require.NoError(t, err)

	cacheCtx, err := db.GetCacheContext()
	require.NoError(t, err)
	require.Equal(t, coins(10), input.BankKeeper.GetAllBalances(cacheCtx, aliasAddr))
	require.Equal(t, coins(34), input.BankKeeper.GetAllBalances(cacheCtx, contractAddr))
	require.Equal(t, coins(8), input.BankKeeper.GetAllBalances(cacheCtx, recipientAddr))

	require.Equal(t, uint64(993), db.GetBalance(sender).Uint64())
	require.Equal(t, uint64(10), db.GetBalance(alias).Uint64(),
		"32-byte Wasm balance events must not overwrite the suffix EOA balance")
	require.Equal(t, uint64(8), db.GetBalance(recipient).Uint64())
}
