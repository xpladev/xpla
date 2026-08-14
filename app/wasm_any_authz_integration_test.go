package app

import (
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdkmock "github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	xplatypes "github.com/xpladev/xpla/types"
)

const canonicalEVMMsgTypeURL = "/cosmos.evm.vm.v1.MsgEthereumTx"

const wasmAnyTestChainID = "wasm-any_37-1"

var (
	wasmAnyTestBlockHash = crypto.Keccak256Hash([]byte("wasm Any integration block"))
	wasmAnyTestLogTopic  = crypto.Keccak256Hash([]byte("wasm Any inner EVM log"))
)

func TestWasmAnyGuard_R3PreseededEOAGrantRejectsMsgExec(t *testing.T) {
	innerKey := mustIntegrationPrivateKey(t)

	t.Run("unguarded control executes the inner EVM transaction", func(t *testing.T) {
		fixture := setupWasmAnyIntegration(t, true)
		defer fixture.close(t)

		innerMsg, innerHash, executeMsg := fixture.seedR3(t, innerKey)
		innerAddress := innerMsg.GetSender()
		createdContract := crypto.CreateAddress(innerAddress, innerMsg.AsTransaction().Nonce())

		_, err := fixture.contractKeeper.Execute(
			fixture.ctx, fixture.wasmContract, fixture.caller, executeMsg, nil,
		)
		require.NoError(t, err)
		require.Equal(t, uint64(1), fixture.app.EvmKeeper.GetNonce(fixture.ctx, innerAddress))
		codeHash := fixture.app.EvmKeeper.GetCodeHash(fixture.ctx, createdContract)
		require.False(t, vmtypes.IsEmptyCodeHash(codeHash.Bytes()))
		require.NotEmpty(t, fixture.app.EvmKeeper.GetCode(fixture.ctx, codeHash))
		require.Equal(
			t,
			common.BigToHash(big.NewInt(42)),
			fixture.app.EvmKeeper.GetState(fixture.ctx, createdContract, common.Hash{}),
		)

		response := fixture.recorder.ethereumResponse(t)
		require.Equal(t, innerHash.Hex(), response.Hash)
		require.Empty(t, response.VmError)
		require.NotZero(t, response.GasUsed)
		require.Equal(t, wasmAnyTestBlockHash.Bytes(), response.BlockHash)
		require.Len(t, response.Logs, 1)
		require.Equal(t, createdContract.Hex(), response.Logs[0].Address)
		require.Equal(t, []string{wasmAnyTestLogTopic.Hex()}, response.Logs[0].Topics)

		receipt := fixture.receipts.recordedReceipt(t)
		require.Equal(t, ethtypes.ReceiptStatusSuccessful, receipt.Status)
		require.Equal(t, createdContract, receipt.ContractAddress)
		require.Equal(t, response.GasUsed, receipt.GasUsed)
		require.Equal(t, big.NewInt(fixture.ctx.BlockHeight()), receipt.BlockNumber)
		require.Equal(t, uint(0), receipt.TransactionIndex)
		require.Len(t, receipt.Logs, 1)
		require.Equal(t, innerHash, receipt.TxHash)
		require.Equal(t, wasmAnyTestBlockHash, receipt.BlockHash)
		require.Equal(t, createdContract, receipt.Logs[0].Address)
		require.Equal(t, []common.Hash{wasmAnyTestLogTopic}, receipt.Logs[0].Topics)
		require.True(t, fixture.recorder.hasSDKEvent(innerHash))

		stored, _ := fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, fixture.wasmContract, sdk.AccAddress(innerAddress.Bytes()), canonicalEVMMsgTypeURL,
		)
		require.NotNil(t, stored)
	})

	t.Run("guarded path exposes no inner EVM surface and preserves the grant", func(t *testing.T) {
		fixture := setupWasmAnyIntegration(t, false)
		defer fixture.close(t)

		innerMsg, innerHash, executeMsg := fixture.seedR3(t, innerKey)
		innerAddress := innerMsg.GetSender()
		createdContract := crypto.CreateAddress(innerAddress, innerMsg.AsTransaction().Nonce())
		nonceBefore := fixture.app.EvmKeeper.GetNonce(fixture.ctx, innerAddress)
		eventsBefore := len(fixture.ctx.EventManager().Events())

		_, err := fixture.contractKeeper.Execute(
			fixture.ctx, fixture.wasmContract, fixture.caller, executeMsg, nil,
		)
		require.ErrorContains(t, err, "found disabled msg type "+canonicalEVMMsgTypeURL)
		require.Equal(t, nonceBefore, fixture.app.EvmKeeper.GetNonce(fixture.ctx, innerAddress))
		codeHash := fixture.app.EvmKeeper.GetCodeHash(fixture.ctx, createdContract)
		require.True(t, vmtypes.IsEmptyCodeHash(codeHash.Bytes()))
		require.Empty(t, fixture.app.EvmKeeper.GetCode(fixture.ctx, codeHash))
		require.Equal(
			t,
			common.Hash{},
			fixture.app.EvmKeeper.GetState(fixture.ctx, createdContract, common.Hash{}),
		)

		for _, event := range fixture.ctx.EventManager().Events()[eventsBefore:] {
			require.NotEqual(t, vmtypes.EventTypeEthereumTx, event.Type)
		}
		require.Nil(t, fixture.receipts.receipt)
		require.False(t, fixture.recorder.hasSDKEvent(innerHash))

		stored, expiration := fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, fixture.wasmContract, sdk.AccAddress(innerAddress.Bytes()), canonicalEVMMsgTypeURL,
		)
		require.NotNil(t, stored)
		require.Nil(t, expiration)
		require.Equal(t, canonicalEVMMsgTypeURL, stored.MsgTypeURL())
	})
}

func TestWasmAnyGuard_R4MsgGrantDoesNotPersist(t *testing.T) {
	t.Run("unguarded signer-valid control persists the canonical grant", func(t *testing.T) {
		fixture := setupWasmAnyIntegration(t, true)
		defer fixture.close(t)

		grantee, executeMsg := fixture.r4MsgGrant(t)
		stored, _ := fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, grantee, fixture.wasmContract, canonicalEVMMsgTypeURL,
		)
		require.Nil(t, stored)

		_, err := fixture.contractKeeper.Execute(
			fixture.ctx, fixture.wasmContract, fixture.caller, executeMsg, nil,
		)
		require.NoError(t, err)
		stored, expiration := fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, grantee, fixture.wasmContract, canonicalEVMMsgTypeURL,
		)
		require.NotNil(t, stored)
		require.Nil(t, expiration)
		require.Equal(t, canonicalEVMMsgTypeURL, stored.MsgTypeURL())
	})

	t.Run("guarded path rejects before persistence", func(t *testing.T) {
		fixture := setupWasmAnyIntegration(t, false)
		defer fixture.close(t)

		grantee, executeMsg := fixture.r4MsgGrant(t)
		stored, _ := fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, grantee, fixture.wasmContract, canonicalEVMMsgTypeURL,
		)
		require.Nil(t, stored)

		_, err := fixture.contractKeeper.Execute(
			fixture.ctx, fixture.wasmContract, fixture.caller, executeMsg, nil,
		)
		require.ErrorContains(t, err, "found disabled msg type "+canonicalEVMMsgTypeURL)
		stored, _ = fixture.app.AuthzKeeper.GetAuthorization(
			fixture.ctx, grantee, fixture.wasmContract, canonicalEVMMsgTypeURL,
		)
		require.Nil(t, stored)
	})
}

type wasmAnyIntegrationFixture struct {
	app            *XplaApp
	ctx            sdk.Context
	contractKeeper *wasmkeeper.PermissionedKeeper
	wasmContract   sdk.AccAddress
	caller         sdk.AccAddress
	recorder       *wasmAnyMessengerRecorder
	receipts       *wasmAnyReceiptRecorder
}

func setupWasmAnyIntegration(t *testing.T, unguarded bool) *wasmAnyIntegrationFixture {
	t.Helper()

	validatorPV := sdkmock.NewPV()
	validatorPubKey, err := validatorPV.GetPubKey()
	require.NoError(t, err)
	sdkValidatorPubKey, err := cryptocodec.FromCmtPubKeyInterface(validatorPubKey)
	require.NoError(t, err)

	xpla := NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		t.TempDir(),
		EmptyAppOptions{},
		EmptyWasmOptions,
	)
	ctx := xpla.BaseApp.NewUncachedContext(false, tmproto.Header{
		Height:          1,
		Time:            time.Unix(1, 0).UTC(),
		ProposerAddress: validatorPubKey.Address(),
	}).WithHeaderHash(wasmAnyTestBlockHash.Bytes())
	validator, err := stakingtypes.NewValidator(
		sdk.ValAddress(sdkValidatorPubKey.Address()).String(),
		sdkValidatorPubKey,
		stakingtypes.Description{},
	)
	require.NoError(t, err)
	require.NoError(t, xpla.StakingKeeper.SetValidator(ctx, validator))
	require.NoError(t, xpla.StakingKeeper.SetValidatorByConsAddr(ctx, validator))
	require.NotNil(t, xpla.AccountKeeper.GetModuleAccount(ctx, minttypes.ModuleName))
	require.NotNil(t, xpla.AccountKeeper.GetModuleAccount(ctx, authtypes.FeeCollectorName))
	require.NoError(t, xpla.WasmKeeper.SetParams(ctx, wasmtypes.DefaultParams()))

	evmParams := vmtypes.DefaultParams()
	evmParams.EvmDenom = xplatypes.DefaultDenom
	evmParams.ExtendedDenomOptions = &vmtypes.ExtendedDenomOptions{ExtendedDenom: xplatypes.DefaultDenom}
	require.NoError(t, xpla.EvmKeeper.SetParams(ctx, evmParams))
	evmCoinInfo := vmtypes.EvmCoinInfo{
		Denom:         xplatypes.DefaultDenom,
		ExtendedDenom: xplatypes.DefaultDenom,
		DisplayDenom:  "xpla",
		Decimals:      uint32(vmtypes.EighteenDecimals),
	}
	require.NoError(t, xpla.EvmKeeper.SetEvmCoinInfo(ctx, evmCoinInfo))
	vmtypes.SetDefaultEvmCoinInfo(evmCoinInfo)
	require.NoError(t, xpla.FeeMarketKeeper.SetParams(ctx, feemarkettypes.DefaultParams()))
	fundWasmAnyModule(
		t, xpla, ctx, authtypes.FeeCollectorName,
		sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(1_000_000_000_000_000_000)),
	)
	receipts := &wasmAnyReceiptRecorder{}
	require.False(t, xpla.EvmKeeper.HasHooks())
	xpla.EvmKeeper.SetHooks(receipts)

	caller := sdk.AccAddress(common.HexToAddress("0x1111111111111111111111111111111111111111").Bytes())
	xpla.AccountKeeper.SetAccount(ctx, xpla.AccountKeeper.NewAccountWithAddress(ctx, caller))

	wasmCode, err := os.ReadFile(filepath.Join(
		"..", "tests", "solidity", "suites", "misc", "any_dispatch.wasm",
	))
	require.NoError(t, err)

	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&xpla.WasmKeeper)
	codeID, _, err := contractKeeper.Create(ctx, caller, wasmCode, nil)
	require.NoError(t, err)
	wasmContract, _, err := contractKeeper.Instantiate(
		ctx, codeID, caller, nil, []byte(`{}`), "wasm Any guard integration", nil,
	)
	require.NoError(t, err)

	selectedMessenger := currentTestWasmMessenger(t, &xpla.WasmKeeper)
	if unguarded {
		selectedMessenger = newUnguardedWasmMessenger(xpla, &xpla.WasmKeeper)
	}
	recorder := &wasmAnyMessengerRecorder{nested: selectedMessenger}
	setTestWasmResponseMessenger(t, &xpla.WasmKeeper, recorder)

	return &wasmAnyIntegrationFixture{
		app:            xpla,
		ctx:            ctx,
		contractKeeper: contractKeeper,
		wasmContract:   wasmContract,
		caller:         caller,
		recorder:       recorder,
		receipts:       receipts,
	}
}

func (f *wasmAnyIntegrationFixture) close(t *testing.T) {
	t.Helper()
	require.NoError(t, f.app.Close())
}

func (f *wasmAnyIntegrationFixture) seedR3(
	t *testing.T,
	innerKey *ecdsa.PrivateKey,
) (*vmtypes.MsgEthereumTx, common.Hash, []byte) {
	t.Helper()
	innerAddress := crypto.PubkeyToAddress(innerKey.PublicKey)
	innerCosmosAddress := sdk.AccAddress(innerAddress.Bytes())
	f.app.AccountKeeper.SetAccount(
		f.ctx, f.app.AccountKeeper.NewAccountWithAddress(f.ctx, innerCosmosAddress),
	)
	fundWasmAnyAccount(
		t, f.app, f.ctx, innerCosmosAddress,
		sdk.NewCoin(xplatypes.DefaultDenom, sdkmath.NewInt(1_000_000_000_000_000_000)),
	)

	grant := authz.NewGenericAuthorization(canonicalEVMMsgTypeURL)
	require.NoError(t, f.app.AuthzKeeper.SaveGrant(
		f.ctx, f.wasmContract, innerCosmosAddress, grant, nil,
	))
	stored, expiration := f.app.AuthzKeeper.GetAuthorization(
		f.ctx, f.wasmContract, innerCosmosAddress, canonicalEVMMsgTypeURL,
	)
	require.NotNil(t, stored)
	require.Nil(t, expiration)
	require.Equal(t, canonicalEVMMsgTypeURL, stored.MsgTypeURL())

	innerMsg, innerHash := signedIntegrationEVMMsg(t, innerKey)
	require.Equal(t, innerAddress, innerMsg.GetSender())
	accepted, err := stored.Accept(f.ctx, innerMsg)
	require.NoError(t, err)
	require.True(t, accepted.Accept)
	require.False(t, accepted.Delete)
	require.Nil(t, accepted.Updated)

	msgExec := authz.NewMsgExec(f.wasmContract, []sdk.Msg{innerMsg})
	return innerMsg, innerHash, wasmDispatchAnyJSON(t, sdk.MsgTypeURL(&msgExec), &msgExec)
}

func (f *wasmAnyIntegrationFixture) r4MsgGrant(t *testing.T) (sdk.AccAddress, []byte) {
	t.Helper()
	grantee := sdk.AccAddress(common.HexToAddress("0x2222222222222222222222222222222222222222").Bytes())
	msgGrant, err := authz.NewMsgGrant(
		f.wasmContract,
		grantee,
		authz.NewGenericAuthorization(canonicalEVMMsgTypeURL),
		nil,
	)
	require.NoError(t, err)
	return grantee, wasmDispatchAnyJSON(t, sdk.MsgTypeURL(msgGrant), msgGrant)
}

// The unguarded control is deliberately confined to this _test.go file. It
// replaces the private response handler in one in-memory test keeper only;
// production construction and runtime configuration expose no bypass.
func setTestWasmResponseMessenger(
	t *testing.T,
	keeper *wasmkeeper.Keeper,
	messenger wasmkeeper.Messenger,
) {
	t.Helper()
	handler := wasmkeeper.NewDefaultWasmVMContractResponseHandler(
		wasmkeeper.NewMessageDispatcher(messenger, keeper),
	)
	setPrivateTestField(t, keeper, "wasmVMResponseHandler", handler)
}

func currentTestWasmMessenger(t *testing.T, keeper *wasmkeeper.Keeper) wasmkeeper.Messenger {
	t.Helper()
	field := privateTestField(t, keeper, "messenger")
	messenger, ok := field.Interface().(wasmkeeper.Messenger)
	require.True(t, ok)
	return messenger
}

func newUnguardedWasmMessenger(xpla *XplaApp, keeper *wasmkeeper.Keeper) wasmkeeper.Messenger {
	return wasmkeeper.NewDefaultMessageHandler(
		keeper,
		xpla.MsgServiceRouter(),
		xpla.IBCKeeper.ChannelKeeper,
		xpla.IBCKeeper.ChannelKeeper,
		&xpla.BankKeeper,
		xpla.appCodec,
		xpla.TransferKeeper,
	)
}

func fundWasmAnyAccount(
	t *testing.T,
	xpla *XplaApp,
	ctx sdk.Context,
	address sdk.AccAddress,
	coin sdk.Coin,
) {
	t.Helper()
	coins := sdk.NewCoins(coin)
	require.NoError(t, xpla.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, xpla.BankKeeper.SendCoinsFromModuleToAccount(ctx, minttypes.ModuleName, address, coins))
}

func fundWasmAnyModule(
	t *testing.T,
	xpla *XplaApp,
	ctx sdk.Context,
	module string,
	coin sdk.Coin,
) {
	t.Helper()
	coins := sdk.NewCoins(coin)
	require.NoError(t, xpla.BankKeeper.MintCoins(ctx, minttypes.ModuleName, coins))
	require.NoError(t, xpla.BankKeeper.SendCoinsFromModuleToModule(ctx, minttypes.ModuleName, module, coins))
}

func setPrivateTestField(t *testing.T, target any, name string, value any) {
	t.Helper()
	field := privateTestField(t, target, name)
	field.Set(reflect.ValueOf(value))
}

func privateTestField(t *testing.T, target any, name string) reflect.Value {
	t.Helper()
	field := reflect.ValueOf(target).Elem().FieldByName(name)
	require.True(t, field.IsValid(), "missing private test field %s", name)
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem() //nolint:gosec
}

type wasmAnyMessengerRecorder struct {
	nested         wasmkeeper.Messenger
	prepareContext func(sdk.Context) sdk.Context
	dispatches     []wasmAnyDispatch
}

type wasmAnyDispatch struct {
	events       []sdk.Event
	data         [][]byte
	msgResponses [][]*codectypes.Any
	err          error
}

func (r *wasmAnyMessengerRecorder) DispatchMsg(
	ctx sdk.Context,
	contractAddr sdk.AccAddress,
	contractIBCPortID string,
	msg wasmvmtypes.CosmosMsg,
) ([]sdk.Event, [][]byte, [][]*codectypes.Any, error) {
	if r.prepareContext != nil {
		ctx = r.prepareContext(ctx)
	}
	events, data, responses, err := r.nested.DispatchMsg(ctx, contractAddr, contractIBCPortID, msg)
	r.dispatches = append(r.dispatches, wasmAnyDispatch{
		events: events, data: data, msgResponses: responses, err: err,
	})
	return events, data, responses, err
}

func (r *wasmAnyMessengerRecorder) ethereumResponse(t *testing.T) *vmtypes.MsgEthereumTxResponse {
	t.Helper()
	require.Len(t, r.dispatches, 1)
	dispatch := r.dispatches[0]
	require.NoError(t, dispatch.err)
	require.Len(t, dispatch.data, 1)

	var authzResponse authz.MsgExecResponse
	require.NoError(t, proto.Unmarshal(dispatch.data[0], &authzResponse))
	require.Len(t, authzResponse.Results, 1)
	var evmResponse vmtypes.MsgEthereumTxResponse
	require.NoError(t, proto.Unmarshal(authzResponse.Results[0], &evmResponse))
	return &evmResponse
}

func (r *wasmAnyMessengerRecorder) hasSDKEvent(hash common.Hash) bool {
	for _, dispatch := range r.dispatches {
		for _, event := range dispatch.events {
			if event.Type != vmtypes.EventTypeEthereumTx {
				continue
			}
			for _, attribute := range event.Attributes {
				if attribute.Key == vmtypes.AttributeKeyEthereumTxHash && common.HexToHash(attribute.Value) == hash {
					return true
				}
			}
		}
	}
	return false
}

type wasmAnyReceiptRecorder struct {
	receipt *ethtypes.Receipt
}

func (r *wasmAnyReceiptRecorder) recordedReceipt(t *testing.T) *ethtypes.Receipt {
	t.Helper()
	require.NotNil(t, r.receipt)
	return r.receipt
}

func (r *wasmAnyReceiptRecorder) PostTxProcessing(
	_ sdk.Context,
	_ common.Address,
	_ core.Message,
	receipt *ethtypes.Receipt,
) error {
	if r.receipt != nil {
		return nil
	}
	copyReceipt := *receipt
	copyReceipt.Logs = append([]*ethtypes.Log(nil), receipt.Logs...)
	r.receipt = &copyReceipt
	return nil
}

func signedIntegrationEVMMsg(t *testing.T, key *ecdsa.PrivateKey) (*vmtypes.MsgEthereumTx, common.Hash) {
	return signedIntegrationEVMMsgAtNonce(t, key, 0)
}

func signedIntegrationEVMMsgAtNonce(
	t *testing.T,
	key *ecdsa.PrivateKey,
	nonce uint64,
) (*vmtypes.MsgEthereumTx, common.Hash) {
	t.Helper()
	chainID := new(big.Int).Set(vmtypes.GetEthChainConfig().ChainID)
	signer := ethtypes.LatestSignerForChainID(chainID)
	tx, err := ethtypes.SignNewTx(key, signer, &ethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1_000_000_000),
		Gas:      500_000,
		Value:    big.NewInt(0),
		Data:     evmCreateCode(wasmAnyTestLogTopic),
	})
	require.NoError(t, err)

	var msg vmtypes.MsgEthereumTx
	require.NoError(t, msg.FromSignedEthereumTx(tx, signer))
	require.NoError(t, msg.ValidateBasic())
	return &msg, tx.Hash()
}

func evmCreateCode(topic common.Hash) []byte {
	code := []byte{0x60, 0x2a, 0x60, 0x00, 0x55, 0x7f}
	code = append(code, topic.Bytes()...)
	code = append(code,
		0x60, 0x00, 0x60, 0x00, 0xa1,
		0x60, 0x05, 0x60, 0x37, 0x60, 0x00, 0x39,
		0x60, 0x05, 0x60, 0x00, 0xf3,
		0x60, 0x00, 0x60, 0x00, 0xf3,
	)
	return code
}

func mustIntegrationPrivateKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA("3b7955d25189c99a7468192fcbc6429205c158834053ebe3f78f4512ab432db9")
	require.NoError(t, err)
	return key
}

func wasmDispatchAnyJSON(t *testing.T, typeURL string, msg proto.Message) []byte {
	t.Helper()
	value, err := proto.Marshal(msg)
	require.NoError(t, err)
	executeMsg, err := json.Marshal(map[string]any{
		"dispatch_any": map[string]any{
			"type_url": typeURL,
			"value":    value,
		},
	})
	require.NoError(t, err)
	return executeMsg
}
