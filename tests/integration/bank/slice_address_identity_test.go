package bank_test

import (
	"bytes"
	"math/big"
	"testing"

	"cosmossdk.io/collections"
	ccodec "cosmossdk.io/collections/codec"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
	xbankkeeper "github.com/xpladev/xpla/x/bank/keeper"
)

func TestBankRecipientCreationPreservesExactEVMIdentity(t *testing.T) {
	for _, order := range []string{"eoa-first", "long-first"} {
		t.Run(order, func(t *testing.T) {
			input := testutil.CreateTestInput(t)
			sender := sdk.AccAddress(testutil.Pks[0].Address())
			eoa := sdk.AccAddress(bytes.Repeat([]byte{0x33}, 20))
			long := sdk.AccAddress(append(bytes.Repeat([]byte{0x44}, 12), eoa...))
			require.NoError(t, input.InitAccountWithCoins(sender, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(100)))))
			msgServer := xbankkeeper.NewMsgServerImpl(input.BankKeeper)

			createEOA := func() {
				account := input.AccountKeeper.NewAccountWithAddress(input.Ctx, eoa)
				require.NoError(t, account.SetSequence(23))
				input.AccountKeeper.SetAccount(input.Ctx, account)
			}
			send := func(address sdk.AccAddress, amount int64) {
				_, err := msgServer.Send(input.Ctx, &banktypes.MsgSend{
					FromAddress: sender.String(),
					ToAddress:   address.String(),
					Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(amount))),
				})
				require.NoError(t, err)
			}

			if order == "eoa-first" {
				createEOA()
				send(eoa, 2)
				send(long, 3)
			} else {
				send(long, 3)
				createEOA()
				send(eoa, 2)
			}

			require.Equal(t, eoa, input.AccountKeeper.GetAccount(input.Ctx, eoa).GetAddress())
			require.Equal(t, long, input.AccountKeeper.GetAccount(input.Ctx, long).GetAddress())
			require.Equal(t, uint64(23), input.App.EvmKeeper.GetNonce(input.Ctx, common.BytesToAddress(eoa)))
			require.Equal(t, sdkmath.NewInt(2), input.BankKeeper.GetBalance(input.Ctx, eoa, sdk.DefaultBondDenom).Amount)
			require.Equal(t, sdkmath.NewInt(3), input.BankKeeper.GetBalance(input.Ctx, long, sdk.DefaultBondDenom).Amount)
			_, hasAlias, err := input.App.WasmKeeper.ResolveWasmAlias(input.Ctx, eoa)
			require.NoError(t, err)
			require.False(t, hasAlias)
		})
	}
}

func TestCollidingLongAccountCannotReenableStaleEVMNonce(t *testing.T) {
	input := testutil.CreateTestInput(t)
	key, err := crypto.HexToECDSA("59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d")
	require.NoError(t, err)
	sender := crypto.PubkeyToAddress(key.PublicKey)
	eoa := sdk.AccAddress(sender.Bytes())
	long := sdk.AccAddress(append(bytes.Repeat([]byte{0x67}, 12), eoa...))
	account := input.AccountKeeper.NewAccountWithAddress(input.Ctx, eoa)
	require.NoError(t, account.SetSequence(5))
	input.AccountKeeper.SetAccount(input.Ctx, account)
	funds := sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(1_000_000_000_000_000_000)))
	require.NoError(t, input.InitAccountWithCoins(eoa, funds))

	bankServer := xbankkeeper.NewMsgServerImpl(input.BankKeeper)
	_, err = bankServer.Send(input.Ctx, &banktypes.MsgSend{
		FromAddress: eoa.String(),
		ToAddress:   long.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.OneInt())),
	})
	require.NoError(t, err)
	require.Equal(t, uint64(5), input.App.EvmKeeper.GetNonce(input.Ctx, sender))

	receipts := &testutil.EVMReceiptRecorder{}
	require.False(t, input.App.EvmKeeper.HasHooks())
	input.App.EvmKeeper.SetHooks(receipts)
	to := common.HexToAddress("0x0000000000000000000000000000000000009876")
	signer := ethtypes.LatestSignerForChainID(vmtypes.GetEthChainConfig().ChainID)
	tx, err := ethtypes.SignNewTx(key, signer, &ethtypes.LegacyTx{
		Nonce: 4, GasPrice: big.NewInt(1_000_000_000), Gas: 200_000, To: &to,
	})
	require.NoError(t, err)
	var msg vmtypes.MsgEthereumTx
	require.NoError(t, msg.FromSignedEthereumTx(tx, signer))
	cosmosTx, err := msg.BuildTx(input.App.GetTxConfig().NewTxBuilder(), xplatypes.DefaultDenom)
	require.NoError(t, err)
	txBytes, err := input.App.GetTxConfig().TxEncoder()(cosmosTx)
	require.NoError(t, err)

	balanceBefore := input.BankKeeper.GetBalance(input.Ctx, eoa, xplatypes.DefaultDenom)
	result := input.CommitTransactionResult(t, txBytes)
	require.NotZero(t, result.Code)
	require.Contains(t, result.Log, "invalid nonce")
	ctx := input.EVMContext()
	require.Equal(t, uint64(5), input.App.EvmKeeper.GetNonce(ctx, sender))
	require.Equal(t, balanceBefore, input.BankKeeper.GetBalance(ctx, eoa, xplatypes.DefaultDenom))
	require.Empty(t, result.Events)
	require.Zero(t, receipts.Len())
}

func TestBankRecipientCannotShadowGovModuleOrBypassEVMGuard(t *testing.T) {
	input := testutil.CreateTestInput(t)
	sender := sdk.AccAddress(testutil.Pks[0].Address())
	govAccount := authtypes.NewEmptyModuleAccount(govtypes.ModuleName)
	input.AccountKeeper.NewAccount(input.Ctx, govAccount)
	govAddr := govAccount.GetAddress()
	long := sdk.AccAddress(append(bytes.Repeat([]byte{0x55}, 12), govAddr...))
	require.NoError(t, input.InitAccountWithCoins(sender, sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.NewInt(10)))))
	input.AccountKeeper.SetAccount(input.Ctx, govAccount)

	// A poisoned legacy entry must not affect exact module identity.
	legacyAliases := collections.NewMap(
		collections.NewSchemaBuilder(runtime.NewKVStoreService(input.App.GetKey(authtypes.StoreKey))),
		collections.NewPrefix("sliceAddress"), "legacy_slice_address",
		sdk.AccAddressKey, ccodec.KeyToValueCodec(sdk.AccAddressKey),
	)
	require.NoError(t, legacyAliases.Set(input.Ctx, govAddr, long))
	msgServer := xbankkeeper.NewMsgServerImpl(input.BankKeeper)
	_, err := msgServer.Send(input.Ctx, &banktypes.MsgSend{
		FromAddress: sender.String(),
		ToAddress:   long.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdkmath.OneInt())),
	})
	require.NoError(t, err)

	resolved := input.AccountKeeper.GetAccount(input.Ctx, govAddr)
	_, isModule := resolved.(sdk.ModuleAccountI)
	require.True(t, isModule)
	require.Equal(t, govAddr, resolved.GetAddress())
	require.Equal(t, long, input.AccountKeeper.GetAccount(input.Ctx, long).GetAddress())
	err = input.App.EvmKeeper.SetBalanceWithLocked(
		input.Ctx,
		common.BytesToAddress(govAddr),
		uint256.NewInt(0),
		big.NewInt(0),
	)
	require.ErrorIs(t, err, errors.ErrUnauthorized)
}
