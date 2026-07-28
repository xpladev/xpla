package app

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtbytes "github.com/cometbft/cometbft/libs/bytes"
	cmtquery "github.com/cometbft/cometbft/libs/pubsub/query"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmtrpcclient "github.com/cometbft/cometbft/rpc/client"
	cmtrpcmock "github.com/cometbft/cometbft/rpc/client/mock"
	cmtrpctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/server"
	sdkmock "github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/indexer"
	pcommon "github.com/cosmos/evm/precompiles/common"
	rpcbackend "github.com/cosmos/evm/rpc/backend"
	rpctypes "github.com/cosmos/evm/rpc/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	pwasm "github.com/xpladev/xpla/precompile/wasm"
	xplatypes "github.com/xpladev/xpla/types"
)

const outerWasmAnyGasLimit = uint64(10_000_000)

func TestWasmAnyGuard_R3CommittedOuterEVMSurfaces(t *testing.T) {
	// The EVM Wasm precompile ABI carries a 20-byte address. Exercise a real
	// keeper-instantiated legacy/SDK-length Wasm contract that is reachable from
	// that ABI rather than truncating a modern 32-byte contract address.
	previousContractAddressLength := wasmtypes.ContractAddrLen
	wasmtypes.ContractAddrLen = common.AddressLength
	t.Cleanup(func() { wasmtypes.ContractAddrLen = previousContractAddressLength })

	fixture := setupCommittedWasmAnyIntegration(t)
	defer fixture.close(t)

	unguarded := fixture.deliver(t, 2, 0, 0, fixture.unguardedMessenger)
	require.Equal(t, uint32(0), unguarded.txResult.Code, unguarded.txResult.Log)
	outerResponse := ethereumTxResponse(t, unguarded.txResult)
	require.Empty(t, outerResponse.VmError)
	require.Equal(t, uint64(0), fixture.app.EvmKeeper.GetNonce(unguarded.queryCtx, fixture.innerAddress))
	require.True(t, fixture.receipts.has(unguarded.innerHash))
	require.Len(t, fixture.receipts.get(unguarded.innerHash).Logs, 1)
	require.True(t, hasEthereumEvent(unguarded.txResult.Events, unguarded.innerHash))

	unguardedRPC := newWasmAnyCometRPC(fixture.app, unguarded)
	require.True(t, searchCommittedEthereumEvent(t, unguardedRPC, unguarded.innerHash))
	panicValue := committedIndexerPanic(t, fixture.app, unguarded)
	require.Contains(t, panicValue, "index out of range")
	receiptCount := len(fixture.receipts.receipts)

	guarded := fixture.deliver(t, 3, 1, 0, fixture.guardedMessenger)
	require.Equal(t, uint32(0), guarded.txResult.Code, guarded.txResult.Log)
	guardedResponse := ethereumTxResponse(t, guarded.txResult)
	require.Contains(t, guardedResponse.VmError, "execution reverted")
	require.Equal(t, uint64(0), fixture.app.EvmKeeper.GetNonce(guarded.queryCtx, fixture.innerAddress))
	require.Len(t, fixture.receipts.receipts, receiptCount+1) // outer failed EVM receipt only
	guardedReceipt := fixture.receipts.receipts[len(fixture.receipts.receipts)-1]
	require.Equal(t, guarded.outerHash, guardedReceipt.TxHash)
	require.Equal(t, uint64(ethtypes.ReceiptStatusFailed), guardedReceipt.Status)
	require.False(t, hasEthereumEvent(guarded.txResult.Events, guarded.innerHash))

	guardedSurfaces := queryCommittedWasmAnySurfaces(t, fixture.app, guarded)
	require.NotNil(t, guardedSurfaces.outerReceipt)
	require.NotNil(t, guardedSurfaces.outerTransaction)
	require.True(t, guardedSurfaces.outerBlockMember)
	require.False(t, guardedSurfaces.innerIndexed)
	require.Nil(t, guardedSurfaces.innerTransaction)
	require.Nil(t, guardedSurfaces.innerReceipt)
	require.False(t, guardedSurfaces.innerBlockMember)
	require.False(t, guardedSurfaces.innerLog)
	require.False(t, guardedSurfaces.cosmosEventSearch)

	stored, expiration := fixture.app.AuthzKeeper.GetAuthorization(
		guarded.queryCtx,
		fixture.wasmContract,
		sdk.AccAddress(fixture.innerAddress.Bytes()),
		canonicalEVMMsgTypeURL,
	)
	require.NotNil(t, stored)
	require.Nil(t, expiration)
}

type committedWasmAnyFixture struct {
	app                *XplaApp
	wasmContract       sdk.AccAddress
	outerKey           *ecdsa.PrivateKey
	innerKey           *ecdsa.PrivateKey
	innerAddress       common.Address
	unguardedMessenger wasmkeeper.Messenger
	guardedMessenger   wasmkeeper.Messenger
	precompileKeeper   *wasmkeeper.Keeper
	receipts           *wasmAnyReceiptList
	pendingCtx         sdk.Context
	proposerAddress    []byte
}

type committedWasmAnyBlock struct {
	height      int64
	block       *tmtypes.Block
	blockResult *cmtrpctypes.ResultBlockResults
	txResult    *abci.ExecTxResult
	outerHash   common.Hash
	innerHash   common.Hash
	queryCtx    sdk.Context
}

func setupCommittedWasmAnyIntegration(t *testing.T) *committedWasmAnyFixture {
	t.Helper()

	xpla := NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		t.TempDir(),
		EmptyAppOptions{},
		EmptyWasmOptions,
		baseapp.SetChainID(wasmAnyTestChainID),
	)
	proposerAddress := initializeWasmAnyGenesis(t, xpla)

	ctx := xpla.BaseApp.NewUncachedContext(false, tmproto.Header{
		ChainID: wasmAnyTestChainID,
		Height:  2,
		Time:    time.Unix(2, 0).UTC(),
	})
	precompileKeeper := actualPrecompileWasmKeeper(t, xpla, ctx)

	outerKey := mustPrivateKey(t, "88cbead91aee890d27bf06e003ade3d4e952427e88f88d31d61d3ef5e5d54305")
	innerKey := mustIntegrationPrivateKey(t)
	outerAddress := crypto.PubkeyToAddress(outerKey.PublicKey)
	innerAddress := crypto.PubkeyToAddress(innerKey.PublicKey)
	testBalance, ok := sdkmath.NewIntFromString("20000000000000000000")
	require.True(t, ok)
	for _, address := range []common.Address{outerAddress, innerAddress} {
		accountAddress := sdk.AccAddress(address.Bytes())
		xpla.AccountKeeper.SetAccount(ctx, xpla.AccountKeeper.NewAccountWithAddress(ctx, accountAddress))
		fundWasmAnyAccount(
			t, xpla, ctx, accountAddress,
			sdk.NewCoin(xplatypes.DefaultDenom, testBalance),
		)
	}
	fundWasmAnyModule(
		t, xpla, ctx, authtypes.FeeCollectorName,
		sdk.NewCoin(xplatypes.DefaultDenom, testBalance),
	)

	wasmCode, err := os.ReadFile(filepath.Join(
		"..", "tests", "solidity", "suites", "misc", "any_dispatch.wasm",
	))
	require.NoError(t, err)
	contractKeeper := wasmkeeper.NewDefaultPermissionKeeper(&xpla.WasmKeeper)
	creator := sdk.AccAddress(outerAddress.Bytes())
	codeID, _, err := contractKeeper.Create(ctx, creator, wasmCode, nil)
	require.NoError(t, err)
	wasmContract, _, err := contractKeeper.Instantiate(
		ctx, codeID, creator, nil, []byte(`{}`), "committed wasm Any guard integration", nil,
	)
	require.NoError(t, err)
	require.Len(t, wasmContract, common.AddressLength)
	require.NotNil(t, xpla.AccountKeeper.GetAccount(ctx, wasmContract))

	require.NoError(t, xpla.AuthzKeeper.SaveGrant(
		ctx,
		wasmContract,
		sdk.AccAddress(innerAddress.Bytes()),
		authz.NewGenericAuthorization(canonicalEVMMsgTypeURL),
		nil,
	))

	receipts := &wasmAnyReceiptList{}
	require.False(t, xpla.EvmKeeper.HasHooks())
	xpla.EvmKeeper.SetHooks(receipts)

	return &committedWasmAnyFixture{
		app:                xpla,
		wasmContract:       wasmContract,
		outerKey:           outerKey,
		innerKey:           innerKey,
		innerAddress:       innerAddress,
		unguardedMessenger: newUnguardedWasmMessenger(xpla, precompileKeeper),
		guardedMessenger:   currentTestWasmMessenger(t, precompileKeeper),
		precompileKeeper:   precompileKeeper,
		receipts:           receipts,
		pendingCtx:         ctx,
		proposerAddress:    proposerAddress,
	}
}

// NewAppKeeper returns its keeper aggregate by value after the EVM precompile
// has captured a pointer to the original Wasm keeper. Reach that exact
// production-wired keeper so the test-only control can replace only its
// response messenger; mutating the returned aggregate's copy would not affect
// EVM precompile execution.
func actualPrecompileWasmKeeper(
	t *testing.T,
	xpla *XplaApp,
	ctx sdk.Context,
) *wasmkeeper.Keeper {
	t.Helper()
	params := xpla.EvmKeeper.GetParams(ctx)
	contract, found, err := xpla.EvmKeeper.GetStaticPrecompileInstance(&params, pwasm.Address)
	require.NoError(t, err)
	require.True(t, found)
	precompile, ok := contract.(*pwasm.PrecompiledWasm)
	require.True(t, ok)

	msgServer := privateTestField(t, precompile, "wms").Interface()
	serverValue := reflect.ValueOf(msgServer)
	require.Equal(t, reflect.Pointer, serverValue.Kind())
	keeperField := serverValue.Elem().FieldByName("keeper")
	require.True(t, keeperField.IsValid())
	keeperValue := reflect.NewAt(
		keeperField.Type(), unsafe.Pointer(keeperField.UnsafeAddr()), //nolint:gosec
	).Elem()
	keeper, ok := keeperValue.Interface().(*wasmkeeper.Keeper)
	require.True(t, ok)
	return keeper
}

func initializeWasmAnyGenesis(t *testing.T, xpla *XplaApp) []byte {
	t.Helper()

	validatorPV := sdkmock.NewPV()
	validatorPubKey, err := validatorPV.GetPubKey()
	require.NoError(t, err)
	validatorSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{
		tmtypes.NewValidator(validatorPubKey, 1),
	})
	genesisPV := sdkmock.NewPV()
	genesisPubKey := genesisPV.PrivKey.PubKey()
	genesisAccount := authtypes.NewBaseAccount(
		genesisPubKey.Address().Bytes(), genesisPubKey, 0, 0,
	)
	genesisBalance := banktypes.Balance{
		Address: genesisAccount.GetAddress().String(),
		Coins: sdk.NewCoins(sdk.NewCoin(
			xplatypes.DefaultDenom,
			sdk.DefaultPowerReduction.MulRaw(100),
		)),
	}
	genesis, err := simtestutil.GenesisStateWithValSet(
		xpla.AppCodec(),
		xpla.DefaultGenesis(),
		validatorSet,
		[]authtypes.GenesisAccount{genesisAccount},
		genesisBalance,
	)
	require.NoError(t, err)
	var bankGenesis banktypes.GenesisState
	xpla.AppCodec().MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGenesis)
	bankGenesis.DenomMetadata = append(bankGenesis.DenomMetadata, banktypes.Metadata{
		Base:    xplatypes.DefaultDenom,
		Display: "xpla",
		Name:    "XPLA",
		Symbol:  "XPLA",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: xplatypes.DefaultDenom, Exponent: 0},
			{Denom: "xpla", Exponent: uint32(vmtypes.EighteenDecimals)},
		},
	})
	genesis[banktypes.ModuleName] = xpla.AppCodec().MustMarshalJSON(&bankGenesis)
	var evmGenesis vmtypes.GenesisState
	xpla.AppCodec().MustUnmarshalJSON(genesis[vmtypes.ModuleName], &evmGenesis)
	evmGenesis.Params.EvmDenom = xplatypes.DefaultDenom
	evmGenesis.Params.ExtendedDenomOptions = &vmtypes.ExtendedDenomOptions{
		ExtendedDenom: xplatypes.DefaultDenom,
	}
	evmGenesis.Params.ActiveStaticPrecompiles = append(
		evmGenesis.Params.ActiveStaticPrecompiles,
		pwasm.Address.Hex(),
	)
	genesis[vmtypes.ModuleName] = xpla.AppCodec().MustMarshalJSON(&evmGenesis)
	genesisState, err := json.Marshal(genesis)
	require.NoError(t, err)
	_, err = xpla.InitChain(&abci.RequestInitChain{
		ChainId:       wasmAnyTestChainID,
		AppStateBytes: genesisState,
		ConsensusParams: &tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{MaxBytes: 20_000_000, MaxGas: 100_000_000},
			Validator: &tmproto.ValidatorParams{
				PubKeyTypes: []string{tmtypes.ABCIPubKeyTypeEd25519},
			},
		},
	})
	require.NoError(t, err)
	_, err = xpla.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          1,
		Time:            time.Unix(1, 0).UTC(),
		ProposerAddress: validatorPubKey.Address(),
	})
	require.NoError(t, err)
	_, err = xpla.Commit()
	require.NoError(t, err)
	return append([]byte(nil), validatorPubKey.Address()...)
}

// EVM v0.6 snapshots only persistent KV stores. The unguarded test control
// still needs the real EVM transient bookkeeping to prove which inner surfaces
// would be exposed without the guard, so only that control falls back to the
// enclosing block context for transient keys. Production wiring is unchanged.
type transientFallbackMultiStore struct {
	storetypes.MultiStore
	transient storetypes.MultiStore
}

func newTransientFallbackMultiStore(
	primary storetypes.MultiStore,
	transient storetypes.MultiStore,
) *transientFallbackMultiStore {
	return &transientFallbackMultiStore{MultiStore: primary, transient: transient}
}

func (s *transientFallbackMultiStore) GetStore(key storetypes.StoreKey) storetypes.Store {
	if _, ok := key.(*storetypes.TransientStoreKey); ok {
		return s.transient.GetStore(key)
	}
	return s.MultiStore.GetStore(key)
}

func (s *transientFallbackMultiStore) GetKVStore(key storetypes.StoreKey) storetypes.KVStore {
	if _, ok := key.(*storetypes.TransientStoreKey); ok {
		return s.transient.GetKVStore(key)
	}
	return s.MultiStore.GetKVStore(key)
}

func (f *committedWasmAnyFixture) deliver(
	t *testing.T,
	height int64,
	outerNonce uint64,
	innerNonce uint64,
	messenger wasmkeeper.Messenger,
) committedWasmAnyBlock {
	t.Helper()

	if height == 2 {
		require.Equal(t, height, f.pendingCtx.BlockHeight())
	} else {
		f.pendingCtx = f.app.BaseApp.NewUncachedContext(false, tmproto.Header{
			ChainID: wasmAnyTestChainID,
			Height:  height,
			Time:    time.Unix(height, 0).UTC(),
		})
	}
	recorder := &wasmAnyMessengerRecorder{
		nested: messenger,
		prepareContext: func(ctx sdk.Context) sdk.Context {
			return ctx.WithMultiStore(newTransientFallbackMultiStore(
				ctx.MultiStore(), f.pendingCtx.MultiStore(),
			))
		},
	}
	setTestWasmResponseMessenger(t, f.precompileKeeper, recorder)

	innerMsg, innerHash := signedIntegrationEVMMsgAtNonce(t, f.innerKey, innerNonce)
	msgExec := authz.NewMsgExec(f.wasmContract, []sdk.Msg{innerMsg})
	wasmMsg := wasmDispatchAnyJSON(t, sdk.MsgTypeURL(&msgExec), &msgExec)
	outerAddress := crypto.PubkeyToAddress(f.outerKey.PublicKey)
	calldata, err := pwasm.ABI.Pack(
		string(pwasm.ExecuteContract),
		outerAddress,
		common.BytesToAddress(f.wasmContract.Bytes()),
		wasmMsg,
		[]pcommon.Coin{},
	)
	require.NoError(t, err)
	outerMsg, outerHash := signedOuterWasmPrecompileMsg(t, f.outerKey, outerNonce, calldata)
	cosmosTx, err := outerMsg.BuildTx(f.app.GetTxConfig().NewTxBuilder(), xplatypes.DefaultDenom)
	require.NoError(t, err)
	txBytes, err := f.app.GetTxConfig().TxEncoder()(cosmosTx)
	require.NoError(t, err)

	block := tmtypes.MakeBlock(height, []tmtypes.Tx{txBytes}, nil, nil)
	block.Header.ChainID = wasmAnyTestChainID
	block.Header.Time = time.Unix(height, 0).UTC()
	block.Header.AppHash = f.app.LastCommitID().Hash
	block.Header.ProposerAddress = append([]byte(nil), f.proposerAddress...)
	blockHash := block.Hash()
	response, err := f.app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Txs:             [][]byte{txBytes},
		Hash:            blockHash,
		Height:          height,
		Time:            block.Header.Time,
		ProposerAddress: block.Header.ProposerAddress,
	})
	require.NoError(t, err)
	require.Len(t, response.TxResults, 1)
	_, err = f.app.Commit()
	require.NoError(t, err)

	return committedWasmAnyBlock{
		height: height,
		block:  block,
		blockResult: &cmtrpctypes.ResultBlockResults{
			Height:                height,
			TxsResults:            response.TxResults,
			FinalizeBlockEvents:   response.Events,
			ValidatorUpdates:      response.ValidatorUpdates,
			ConsensusParamUpdates: response.ConsensusParamUpdates,
			AppHash:               f.app.LastCommitID().Hash,
		},
		txResult:  response.TxResults[0],
		outerHash: outerHash,
		innerHash: innerHash,
		queryCtx: f.app.BaseApp.NewUncachedContext(false, tmproto.Header{
			ChainID: wasmAnyTestChainID,
			Height:  height,
			Time:    block.Header.Time,
		}),
	}
}

func (f *committedWasmAnyFixture) close(t *testing.T) {
	t.Helper()
	require.NoError(t, f.app.Close())
}

func signedOuterWasmPrecompileMsg(
	t *testing.T,
	key *ecdsa.PrivateKey,
	nonce uint64,
	calldata []byte,
) (*vmtypes.MsgEthereumTx, common.Hash) {
	t.Helper()
	chainID := new(big.Int).Set(vmtypes.GetEthChainConfig().ChainID)
	signer := ethtypes.LatestSignerForChainID(chainID)
	to := pwasm.Address
	tx, err := ethtypes.SignNewTx(key, signer, &ethtypes.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(1_000_000_000),
		Gas:      outerWasmAnyGasLimit,
		To:       &to,
		Value:    big.NewInt(0),
		Data:     calldata,
	})
	require.NoError(t, err)
	var msg vmtypes.MsgEthereumTx
	require.NoError(t, msg.FromSignedEthereumTx(tx, signer))
	require.NoError(t, msg.ValidateBasic())
	return &msg, tx.Hash()
}

func mustPrivateKey(t *testing.T, value string) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA(value)
	require.NoError(t, err)
	return key
}

type wasmAnyReceiptList struct {
	receipts []*ethtypes.Receipt
}

func (r *wasmAnyReceiptList) PostTxProcessing(
	_ sdk.Context,
	_ common.Address,
	_ core.Message,
	receipt *ethtypes.Receipt,
) error {
	copyReceipt := *receipt
	copyReceipt.Logs = append([]*ethtypes.Log(nil), receipt.Logs...)
	r.receipts = append(r.receipts, &copyReceipt)
	return nil
}

func (r *wasmAnyReceiptList) has(hash common.Hash) bool {
	return r.get(hash) != nil
}

func (r *wasmAnyReceiptList) get(hash common.Hash) *ethtypes.Receipt {
	for _, receipt := range r.receipts {
		if receipt.TxHash == hash {
			return receipt
		}
	}
	return nil
}

func hasEthereumEvent(events []abci.Event, hash common.Hash) bool {
	for _, event := range events {
		if event.Type != vmtypes.EventTypeEthereumTx {
			continue
		}
		for _, attribute := range event.Attributes {
			if attribute.Key == vmtypes.AttributeKeyEthereumTxHash && common.HexToHash(attribute.Value) == hash {
				return true
			}
		}
	}
	return false
}

func ethereumTxResponse(t *testing.T, result *abci.ExecTxResult) vmtypes.MsgEthereumTxResponse {
	t.Helper()
	var txMsgData sdk.TxMsgData
	require.NoError(t, proto.Unmarshal(result.Data, &txMsgData))
	require.Len(t, txMsgData.MsgResponses, 1)
	var response vmtypes.MsgEthereumTxResponse
	require.NoError(t, proto.Unmarshal(txMsgData.MsgResponses[0].Value, &response))
	return response
}

type committedWasmAnySurfaces struct {
	outerReceipt      map[string]interface{}
	innerReceipt      map[string]interface{}
	outerTransaction  *rpctypes.RPCTransaction
	innerTransaction  *rpctypes.RPCTransaction
	outerBlockMember  bool
	innerBlockMember  bool
	innerLog          bool
	logs              [][]*ethtypes.Log
	cosmosEventSearch bool
	innerIndexed      bool
}

func newWasmAnyCometRPC(app *XplaApp, delivered committedWasmAnyBlock) *wasmAnyCometRPC {
	return &wasmAnyCometRPC{
		app: app,
		resultBlock: &cmtrpctypes.ResultBlock{
			BlockID: tmtypes.BlockID{Hash: delivered.block.Hash()},
			Block:   delivered.block,
		},
		blockResult: delivered.blockResult,
	}
}

func committedWasmAnyClientContext(app *XplaApp, rpcClient *wasmAnyCometRPC) client.Context {
	return client.Context{}.
		WithChainID(wasmAnyTestChainID).
		WithCodec(app.AppCodec()).
		WithTxConfig(app.GetTxConfig()).
		WithClient(rpcClient)
}

func committedIndexerPanic(
	t *testing.T,
	app *XplaApp,
	delivered committedWasmAnyBlock,
) (panicText string) {
	t.Helper()
	defer func() {
		if recovered := recover(); recovered != nil {
			panicText = fmt.Sprint(recovered)
		}
	}()
	rpcClient := newWasmAnyCometRPC(app, delivered)
	clientCtx := committedWasmAnyClientContext(app, rpcClient)
	evmIndexer := indexer.NewKVIndexer(dbm.NewMemDB(), log.NewNopLogger(), clientCtx)
	require.NoError(t, evmIndexer.IndexBlock(delivered.block, delivered.blockResult.TxsResults))
	return ""
}

func queryCommittedWasmAnySurfaces(
	t *testing.T,
	app *XplaApp,
	delivered committedWasmAnyBlock,
) committedWasmAnySurfaces {
	t.Helper()

	rpcClient := newWasmAnyCometRPC(app, delivered)
	clientCtx := committedWasmAnyClientContext(app, rpcClient)
	evmIndexer := indexer.NewKVIndexer(dbm.NewMemDB(), log.NewNopLogger(), clientCtx)
	require.NoError(t, evmIndexer.IndexBlock(delivered.block, delivered.blockResult.TxsResults))

	serverCtx := server.NewDefaultContext()
	serverCtx.Viper.Set("evm.evm-chain-id", uint64(37))
	backend := rpcbackend.NewBackend(
		serverCtx, log.NewNopLogger(), clientCtx, false, evmIndexer, app.EVMMempool,
	)

	outerReceipt, err := backend.GetTransactionReceipt(delivered.outerHash)
	require.NoError(t, err)
	outerTransaction, err := backend.GetTransactionByHash(delivered.outerHash)
	require.NoError(t, err)
	logs, err := backend.GetLogs(common.BytesToHash(delivered.block.Hash()))
	require.NoError(t, err)
	block, err := backend.GetBlockByNumber(rpctypes.BlockNumber(delivered.height), true)
	require.NoError(t, err)

	innerResult, innerIndexErr := evmIndexer.GetByTxHash(delivered.innerHash)
	innerIndexed := innerIndexErr == nil
	if innerIndexed {
		require.NotNil(t, innerResult)
	} else {
		require.Nil(t, innerResult)
	}
	innerReceipt, err := backend.GetTransactionReceipt(delivered.innerHash)
	require.NoError(t, err)
	innerTransaction, err := backend.GetTransactionByHash(delivered.innerHash)
	require.NoError(t, err)

	outerBlockMember := rpcBlockHasHash(block, delivered.outerHash)
	innerBlockMember := rpcBlockHasHash(block, delivered.innerHash)
	innerLog := blockLogsHaveHash(logs, delivered.innerHash)
	cosmosEventSearch := searchCommittedEthereumEvent(t, rpcClient, delivered.innerHash)

	return committedWasmAnySurfaces{
		outerReceipt:      outerReceipt,
		innerReceipt:      innerReceipt,
		outerTransaction:  outerTransaction,
		innerTransaction:  innerTransaction,
		outerBlockMember:  outerBlockMember,
		innerBlockMember:  innerBlockMember,
		innerLog:          innerLog,
		logs:              logs,
		cosmosEventSearch: cosmosEventSearch,
		innerIndexed:      innerIndexed,
	}
}

func rpcBlockHasHash(block map[string]interface{}, hash common.Hash) bool {
	txs, ok := block["transactions"].([]interface{})
	if !ok {
		return false
	}
	for _, value := range txs {
		switch tx := value.(type) {
		case *rpctypes.RPCTransaction:
			if tx.Hash == hash {
				return true
			}
		case rpctypes.RPCTransaction:
			if tx.Hash == hash {
				return true
			}
		case common.Hash:
			if tx == hash {
				return true
			}
		}
	}
	return false
}

func blockLogsHaveHash(logs [][]*ethtypes.Log, hash common.Hash) bool {
	for _, txLogs := range logs {
		for _, log := range txLogs {
			if log.TxHash == hash {
				return true
			}
		}
	}
	return false
}

func searchCommittedEthereumEvent(
	t *testing.T,
	rpcClient *wasmAnyCometRPC,
	hash common.Hash,
) bool {
	t.Helper()
	query := vmtypes.EventTypeEthereumTx + "." + vmtypes.AttributeKeyEthereumTxHash + "='" + hash.Hex() + "'"
	results, err := rpcClient.TxSearch(
		context.Background(), query, false, nil, nil, "",
	)
	require.NoError(t, err)
	return len(results.Txs) > 0
}

type wasmAnyCometRPC struct {
	cmtrpcmock.Client
	app         *XplaApp
	resultBlock *cmtrpctypes.ResultBlock
	blockResult *cmtrpctypes.ResultBlockResults
}

var _ cmtrpcclient.Client = (*wasmAnyCometRPC)(nil)

func (c *wasmAnyCometRPC) ABCIQuery(
	ctx context.Context,
	path string,
	data cmtbytes.HexBytes,
) (*cmtrpctypes.ResultABCIQuery, error) {
	return c.ABCIQueryWithOptions(ctx, path, data, cmtrpcclient.DefaultABCIQueryOptions)
}

func (c *wasmAnyCometRPC) ABCIQueryWithOptions(
	ctx context.Context,
	path string,
	data cmtbytes.HexBytes,
	opts cmtrpcclient.ABCIQueryOptions,
) (*cmtrpctypes.ResultABCIQuery, error) {
	response, err := c.app.Query(ctx, &abci.RequestQuery{
		Path:   path,
		Data:   data,
		Height: opts.Height,
		Prove:  opts.Prove,
	})
	if err != nil {
		return nil, err
	}
	return &cmtrpctypes.ResultABCIQuery{Response: *response}, nil
}

func (c *wasmAnyCometRPC) Block(
	_ context.Context,
	_ *int64,
) (*cmtrpctypes.ResultBlock, error) {
	return c.resultBlock, nil
}

func (c *wasmAnyCometRPC) BlockByHash(
	_ context.Context,
	_ []byte,
) (*cmtrpctypes.ResultBlock, error) {
	return c.resultBlock, nil
}

func (c *wasmAnyCometRPC) BlockResults(
	_ context.Context,
	_ *int64,
) (*cmtrpctypes.ResultBlockResults, error) {
	return c.blockResult, nil
}

func (c *wasmAnyCometRPC) ConsensusParams(
	_ context.Context,
	_ *int64,
) (*cmtrpctypes.ResultConsensusParams, error) {
	return &cmtrpctypes.ResultConsensusParams{
		BlockHeight: c.blockResult.Height,
		ConsensusParams: tmtypes.ConsensusParams{
			Block: tmtypes.BlockParams{MaxBytes: 20_000_000, MaxGas: 100_000_000},
		},
	}, nil
}

func (c *wasmAnyCometRPC) UnconfirmedTxs(
	context.Context,
	*int,
) (*cmtrpctypes.ResultUnconfirmedTxs, error) {
	return &cmtrpctypes.ResultUnconfirmedTxs{}, nil
}

func (c *wasmAnyCometRPC) TxSearch(
	_ context.Context,
	query string,
	_ bool,
	_, _ *int,
	_ string,
) (*cmtrpctypes.ResultTxSearch, error) {
	compiled, err := cmtquery.New(query)
	if err != nil {
		return nil, err
	}
	flattened := make(map[string][]string)
	for _, event := range c.blockResult.TxsResults[0].Events {
		for _, attribute := range event.Attributes {
			flattened[event.Type+"."+attribute.Key] = append(
				flattened[event.Type+"."+attribute.Key], attribute.Value,
			)
		}
	}
	matched, err := compiled.Matches(flattened)
	if err != nil || !matched {
		return &cmtrpctypes.ResultTxSearch{}, err
	}
	result := &cmtrpctypes.ResultTx{
		Hash:     c.resultBlock.Block.Txs[0].Hash(),
		Height:   c.blockResult.Height,
		Index:    0,
		TxResult: *c.blockResult.TxsResults[0],
		Tx:       c.resultBlock.Block.Txs[0],
	}
	return &cmtrpctypes.ResultTxSearch{Txs: []*cmtrpctypes.ResultTx{result}, TotalCount: 1}, nil
}
