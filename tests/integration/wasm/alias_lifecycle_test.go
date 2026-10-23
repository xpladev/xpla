package wasm_test

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/base64"
	"fmt"
	"os"
	"testing"

	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

func TestWasmAliasLifecycleThroughProductionRoutes(t *testing.T) {
	if runInIsolatedProcess(t) {
		return
	}

	input := testutil.CreateTestInput(t)
	require.NoError(t, input.App.WasmKeeper.SetParams(input.Ctx, wasmtypes.DefaultParams()))
	key, err := crypto.HexToECDSA("8b3a350cf5c34c9194ca3a545d100b37f74b28dc2a1d26b4d9abf2d1f7a21d5e")
	require.NoError(t, err)
	creator := sdk.AccAddress(crypto.PubkeyToAddress(key.PublicKey).Bytes())
	require.NoError(t, input.InitAccountWithCoins(creator, sdk.NewCoins(
		sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewIntFromUint64(10_000_000_000_000_000_000)),
	)))

	wasmCode, err := os.ReadFile("../../solidity/suites/misc/any_dispatch.wasm")
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&input.App.WasmKeeper)
	codeID, checksum, err := contractKeeper.Create(input.Ctx, creator, wasmCode, nil)
	require.NoError(t, err)

	instantiate := &wasmtypes.MsgInstantiateContract{
		Sender: creator.String(), CodeID: codeID, Label: "routed instantiate", Msg: []byte(`{}`),
	}
	result := commitCosmosMsg(t, &input, key, instantiate)
	require.Zero(t, result.Code, result.Log)
	contract := requireNewContract(t, result.Events, nil)
	requireRegisteredAlias(t, &input, contract)

	salt := []byte("top-level-instantiate2")
	instantiate2 := &wasmtypes.MsgInstantiateContract2{
		Sender: creator.String(), CodeID: codeID, Label: "routed instantiate2", Msg: []byte(`{}`), Salt: salt, FixMsg: true,
	}
	result = commitCosmosMsg(t, &input, key, instantiate2)
	require.Zero(t, result.Code, result.Log)
	contract2 := wasmkeeper.BuildContractAddressPredictable(checksum, creator, salt, []byte(`{}`))
	require.NotNil(t, input.App.WasmKeeper.GetContractInfo(input.EVMContext(), contract2))
	requireRegisteredAlias(t, &input, contract2)

	t.Run("transaction router rolls back registration failure", func(t *testing.T) {
		failureSalt := []byte("top-level-conflict")
		predicted := wasmkeeper.BuildContractAddressPredictable(checksum, creator, failureSalt, []byte(`{}`))
		conflictTarget := sdk.AccAddress(append(bytes.Repeat([]byte{0x81}, 12), predicted[len(predicted)-20:]...))
		ctx := input.EVMContext()
		input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, conflictTarget))
		require.NoError(t, input.App.WasmKeeper.RegisterWasmAlias(ctx, predicted[len(predicted)-20:], conflictTarget))

		result := commitCosmosMsg(t, &input, key, &wasmtypes.MsgInstantiateContract2{
			Sender: creator.String(), CodeID: codeID, Label: "must rollback", Msg: []byte(`{}`), Salt: failureSalt, FixMsg: true,
		})
		require.NotZero(t, result.Code)
		ctx = input.EVMContext()
		require.Nil(t, input.App.WasmKeeper.GetContractInfo(ctx, predicted))
		require.Nil(t, input.AccountKeeper.GetAccount(ctx, predicted))
		resolved, found, err := input.App.WasmKeeper.ResolveWasmAlias(ctx, predicted[len(predicted)-20:])
		require.NoError(t, err)
		require.True(t, found)
		require.Equal(t, conflictTarget, resolved)
	})

	t.Run("nested router registration and rollback", func(t *testing.T) {
		failureSalt := []byte("nested-conflict")
		predicted := wasmkeeper.BuildContractAddressPredictable(checksum, contract, failureSalt, []byte(`{}`))
		conflictTarget := sdk.AccAddress(append(bytes.Repeat([]byte{0x82}, 12), predicted[len(predicted)-20:]...))
		ctx := input.EVMContext()
		input.AccountKeeper.SetAccount(ctx, input.AccountKeeper.NewAccountWithAddress(ctx, conflictTarget))
		require.NoError(t, input.App.WasmKeeper.RegisterWasmAlias(ctx, predicted[len(predicted)-20:], conflictTarget))

		nested := &wasmtypes.MsgInstantiateContract2{
			Sender: contract.String(), CodeID: codeID, Label: "nested rollback", Msg: []byte(`{}`), Salt: failureSalt, FixMsg: true,
		}
		result := commitCosmosMsg(t, &input, key, dispatchAny(t, creator, contract, nested))
		require.NotZero(t, result.Code)
		ctx = input.EVMContext()
		require.Nil(t, input.App.WasmKeeper.GetContractInfo(ctx, predicted))
		require.Nil(t, input.AccountKeeper.GetAccount(ctx, predicted))

		successSalt := []byte("nested-success")
		nested = &wasmtypes.MsgInstantiateContract2{
			Sender: contract.String(), CodeID: codeID, Label: "nested success", Msg: []byte(`{}`), Salt: successSalt, FixMsg: true,
		}
		result = commitCosmosMsg(t, &input, key, dispatchAny(t, creator, contract, nested))
		require.Zero(t, result.Code, result.Log)
		child := wasmkeeper.BuildContractAddressPredictable(checksum, contract, successSalt, []byte(`{}`))
		require.NotNil(t, input.App.WasmKeeper.GetContractInfo(input.EVMContext(), child))
		requireRegisteredAlias(t, &input, child)
	})

	t.Run("store and instantiate uses decorated router service", func(t *testing.T) {
		before := contractSet(input.EVMContext(), &input.App.WasmKeeper.Keeper)
		message := &wasmtypes.MsgStoreAndInstantiateContract{
			Authority:    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
			WASMByteCode: wasmCode,
			Label:        "store and instantiate route",
			Msg:          []byte(`{}`),
		}
		handler := input.App.MsgServiceRouter().Handler(message)
		require.NotNil(t, handler)
		ctx := input.EVMContext()
		cacheCtx, write := ctx.CacheContext()
		_, err := handler(cacheCtx, message)
		require.NoError(t, err)
		write()

		after := contractSet(ctx, &input.App.WasmKeeper.Keeper)
		var created sdk.AccAddress
		for address := range after {
			if _, existed := before[address]; !existed {
				created = sdk.AccAddress(address)
			}
		}
		require.Len(t, created, 32)
		requireRegisteredAlias(t, &input, created)
	})
}

func commitCosmosMsg(t *testing.T, input *testutil.TestInput, key *ecdsa.PrivateKey, message sdk.Msg) *abci.ExecTxResult {
	t.Helper()
	ctx := input.EVMContext()
	address := sdk.AccAddress(crypto.PubkeyToAddress(key.PublicKey).Bytes())
	account := input.App.AccountKeeper.GetAccount(ctx, address)
	require.NotNil(t, account)
	builder := input.App.GetTxConfig().NewTxBuilder()
	require.NoError(t, builder.SetMsgs(message))
	builder.SetGasLimit(testutil.DefaultEVMGasLimit)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(10_000_000_000_000_000))))
	privateKey := &ethsecp256k1.PrivKey{Key: crypto.FromECDSA(key)}
	mode := signing.SignMode_SIGN_MODE_DIRECT
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey:   privateKey.PubKey(),
		Data:     &signing.SingleSignatureData{SignMode: mode},
		Sequence: account.GetSequence(),
	}))
	signature, err := clienttx.SignWithPrivKey(ctx, mode, authsigning.SignerData{
		ChainID:       testutil.TestChainID,
		AccountNumber: account.GetAccountNumber(),
		Sequence:      account.GetSequence(),
	}, builder, privateKey, input.App.GetTxConfig(), account.GetSequence())
	require.NoError(t, err)
	require.NoError(t, builder.SetSignatures(signature))
	txBytes, err := input.App.GetTxConfig().TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return input.CommitTransactionResult(t, txBytes)
}

func dispatchAny(t *testing.T, sender, contract sdk.AccAddress, message proto.Message) *wasmtypes.MsgExecuteContract {
	t.Helper()
	value, err := proto.Marshal(message)
	require.NoError(t, err)
	payload := []byte(fmt.Sprintf(`{"dispatch_any":{"type_url":%q,"value":%q}}`, sdk.MsgTypeURL(message), base64.StdEncoding.EncodeToString(value)))
	return &wasmtypes.MsgExecuteContract{Sender: sender.String(), Contract: contract.String(), Msg: payload}
}

func requireRegisteredAlias(t *testing.T, input *testutil.TestInput, contract sdk.AccAddress) {
	t.Helper()
	resolved, found, err := input.App.WasmKeeper.ResolveWasmAlias(input.EVMContext(), contract[len(contract)-20:])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, contract, resolved)
}

func requireNewContract(t *testing.T, events []abci.Event, excluded sdk.AccAddress) sdk.AccAddress {
	t.Helper()
	for _, event := range events {
		for _, attribute := range event.Attributes {
			if attribute.Key != "_contract_address" {
				continue
			}
			address, err := sdk.AccAddressFromBech32(attribute.Value)
			if err == nil && len(address) == 32 && !address.Equals(excluded) {
				return address
			}
		}
	}
	t.Fatal("contract address event not found")
	return nil
}

func contractSet(ctx context.Context, keeper *wasmkeeper.Keeper) map[string]struct{} {
	result := map[string]struct{}{}
	keeper.IterateContractInfo(ctx, func(address sdk.AccAddress, _ wasmtypes.ContractInfo) bool {
		result[string(address)] = struct{}{}
		return false
	})
	return result
}
