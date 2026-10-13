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

// nestedPrecompileBankKeeper runs the outer bank send inside a second bank
// precompile. The outer balance handler must apply the inner transfer once.
type nestedPrecompileBankKeeper struct {
	pbank.BankKeeper

	evm       *vm.EVM
	caller    common.Address
	contract  sdk.AccAddress
	recipient sdk.AccAddress
	denom     string
	budget    uint64
}

func (k nestedPrecompileBankKeeper) SendCoins(context.Context, sdk.AccAddress, sdk.AccAddress, sdk.Coins) error {
	inner := pbank.NewPrecompiledBank(redirectBankKeeper{
		BankKeeper: k.BankKeeper,
		from:       k.contract,
		to:         k.recipient,
		amount:     sdk.NewCoins(sdk.NewInt64Coin(k.denom, 7)),
	})
	contract := vm.NewContract(k.caller, pbank.Address, uint256.NewInt(0), k.budget, nil)
	input, err := pbank.ABI.Pack(
		"send",
		k.caller,
		common.BytesToAddress(k.recipient.Bytes()),
		cmn.NewCoinsResponse(sdk.NewCoins(sdk.NewInt64Coin(k.denom, 7))),
	)
	if err != nil {
		return err
	}
	contract.Input = input
	_, err = inner.Run(k.evm, contract, false)
	return err
}

// redirectBankKeeper performs the native transfer that the inner precompile
// call stands in for. The 32-byte contract cannot pass the ABI signer check.
type redirectBankKeeper struct {
	pbank.BankKeeper

	from   sdk.AccAddress
	to     sdk.AccAddress
	amount sdk.Coins
}

func (k redirectBankKeeper) SendCoins(ctx context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return k.BankKeeper.SendCoins(ctx, k.from, k.to, k.amount)
}

func TestBankPrecompileNestedPrecompileEventsApplyBalanceOnce(t *testing.T) {
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
	supply := input.BankKeeper.GetSupply(input.Ctx, xplatypes.DefaultDenom)

	const budget = uint64(1_000_000)
	keeper := &nestedPrecompileBankKeeper{
		BankKeeper: input.BankKeeper,
		caller:     sender,
		contract:   contractAddr,
		recipient:  recipientAddr,
		denom:      xplatypes.DefaultDenom,
		budget:     budget,
	}
	precompile := pbank.NewPrecompiledBank(keeper)

	ctx := input.Ctx.WithGasMeter(storetypes.NewGasMeter(budget)).WithEventManager(sdk.NewEventManager())
	db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
	require.Equal(t, uint64(1000), db.GetBalance(sender).Uint64())
	require.Equal(t, uint64(10), db.GetBalance(alias).Uint64())
	require.Equal(t, uint64(5), db.GetBalance(recipient).Uint64())
	require.False(t, input.BankKeeper.BlockedAddr(contractAddr))

	evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(1)}, db, vmtypes.GetEthChainConfig(), vm.Config{})
	evm.Origin = sender
	keeper.evm = evm
	contract := vm.NewContract(sender, pbank.Address, uint256.NewInt(0), budget, nil)
	var err error
	contract.Input, err = pbank.ABI.Pack("send", sender, outerRecipient, cmn.NewCoinsResponse(coins(1)))
	require.NoError(t, err)

	_, err = precompile.Run(evm, contract, false)
	require.NoError(t, err)

	cacheCtx, err := db.GetCacheContext()
	require.NoError(t, err)
	require.Equal(t, coins(10), input.BankKeeper.GetAllBalances(cacheCtx, aliasAddr))
	require.Equal(t, coins(23), input.BankKeeper.GetAllBalances(cacheCtx, contractAddr))
	require.Equal(t, coins(12), input.BankKeeper.GetAllBalances(cacheCtx, recipientAddr))

	require.Equal(t, uint64(1000), db.GetBalance(sender).Uint64())
	require.Equal(t, uint64(10), db.GetBalance(alias).Uint64(),
		"32-byte Wasm balance events must not overwrite the suffix EOA balance")
	require.Equal(t, uint64(12), db.GetBalance(recipient).Uint64(),
		"nested precompile events must credit the recipient once")

	require.NoError(t, db.Commit())
	require.Equal(t, coins(1000), input.BankKeeper.GetAllBalances(ctx, senderAddr))
	require.Equal(t, coins(10), input.BankKeeper.GetAllBalances(ctx, aliasAddr))
	require.Equal(t, coins(23), input.BankKeeper.GetAllBalances(ctx, contractAddr))
	require.Equal(t, coins(12), input.BankKeeper.GetAllBalances(ctx, recipientAddr))
	require.Equal(t, supply, input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom))
}
