package wasm_test

import (
	"crypto/ecdsa"
	"encoding/json"
	"os"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/collections"
	"cosmossdk.io/log"
	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdkmock "github.com/cosmos/cosmos-sdk/testutil/mock"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	xplaapp "github.com/xpladev/xpla/app"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

const genesisImportFixtureEnv = "XPLA_WASM_GENESIS_IMPORT_FIXTURE"

type genesisImportFixture struct {
	AppState               json.RawMessage         `json:"app_state"`
	ConsensusParams        tmproto.ConsensusParams `json:"consensus_params"`
	Height                 int64                   `json:"height"`
	ExpectedValidatorCount int                     `json:"expected_validator_count"`
	Contract               []byte                  `json:"contract"`
	Proposer               []byte                  `json:"proposer"`
}

func TestWasmAliasesGenesisImportAndOrdinaryRestart(t *testing.T) {
	if runInIsolatedProcess(t) {
		return
	}
	if fixturePath := os.Getenv(genesisImportFixtureEnv); fixturePath != "" {
		assertGenesisImportThroughInitChain(t, fixturePath)
		return
	}

	key := persistentWasmKey(t)
	databaseDir := t.TempDir()
	home := t.TempDir()
	database, err := dbm.NewDB("application", dbm.GoLevelDBBackend, databaseDir)
	require.NoError(t, err)
	sourceOption, cleanupSource := captureWasmEngine(t)
	source, proposer := newInitializedWasmApp(t, database, home, key, sourceOption)
	contract := storeAndInstantiateRoutedContract(t, source, proposer, key)
	requireRegisteredAppAlias(t, source, contract)

	exported, err := source.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	height := source.LastBlockHeight()
	cleanupSource()
	require.NoError(t, source.Close())

	database, err = dbm.NewDB("application", dbm.GoLevelDBBackend, databaseDir)
	require.NoError(t, err)
	restartOption, cleanupRestart := captureWasmEngine(t)
	restarted := xplaapp.NewXplaApp(
		log.NewNopLogger(), database, nil, true, map[int64]bool{}, home,
		xplaapp.EmptyAppOptions{}, []wasmkeeper.Option{restartOption}, baseapp.SetChainID(testutil.TestChainID),
	)
	require.Equal(t, height, restarted.LastBlockHeight())
	requireRegisteredAppAlias(t, restarted, contract)
	require.NotNil(t, restarted.WasmKeeper.GetContractInfo(appContext(restarted, proposer), contract))
	cleanupRestart()
	require.NoError(t, restarted.Close())

	fixtureBytes, err := json.Marshal(genesisImportFixture{
		AppState:               exported.AppState,
		ConsensusParams:        exported.ConsensusParams,
		Height:                 exported.Height,
		ExpectedValidatorCount: len(exported.Validators),
		Contract:               append([]byte(nil), contract...),
		Proposer:               append([]byte(nil), proposer...),
	})
	require.NoError(t, err)
	fixturePath := t.TempDir() + "/genesis-import.json"
	require.NoError(t, os.WriteFile(fixturePath, fixtureBytes, 0o600))
	runTestInSubprocess(t, genesisImportFixtureEnv, fixturePath)
}

func assertGenesisImportThroughInitChain(t *testing.T, fixturePath string) {
	t.Helper()
	fixtureBytes, err := os.ReadFile(fixturePath)
	require.NoError(t, err)
	var fixture genesisImportFixture
	require.NoError(t, json.Unmarshal(fixtureBytes, &fixture))

	importOption, cleanupImport := captureWasmEngine(t)
	destination := xplaapp.NewXplaApp(
		log.NewNopLogger(), dbm.NewMemDB(), nil, true, map[int64]bool{}, t.TempDir(),
		xplaapp.EmptyAppOptions{}, []wasmkeeper.Option{importOption}, baseapp.SetChainID(testutil.TestChainID),
	)
	t.Cleanup(func() {
		cleanupImport()
		require.NoError(t, destination.Close())
	})
	genesisTime := time.Unix(fixture.Height, 0).UTC()
	response, err := destination.InitChain(&abci.RequestInitChain{
		Time:            genesisTime,
		ChainId:         testutil.TestChainID,
		ConsensusParams: &fixture.ConsensusParams,
		AppStateBytes:   fixture.AppState,
		InitialHeight:   fixture.Height,
	})
	require.NoError(t, err)
	require.Len(t, response.Validators, fixture.ExpectedValidatorCount)
	_, err = destination.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height:          fixture.Height,
		Time:            genesisTime,
		ProposerAddress: fixture.Proposer,
	})
	require.NoError(t, err)
	_, err = destination.Commit()
	require.NoError(t, err)
	require.Equal(t, fixture.Height, destination.LastBlockHeight())

	contract := sdk.AccAddress(fixture.Contract)
	ctx := appContext(destination, fixture.Proposer)
	requireRegisteredAppAlias(t, destination, contract)
	require.NotNil(t, destination.WasmKeeper.GetContractInfo(ctx, contract))
}

func newInitializedWasmApp(t *testing.T, database dbm.DB, home string, key *ecdsa.PrivateKey, wasmOptions ...wasmkeeper.Option) (*xplaapp.XplaApp, []byte) {
	t.Helper()
	app := xplaapp.NewXplaApp(
		log.NewNopLogger(), database, nil, true, map[int64]bool{}, home,
		xplaapp.EmptyAppOptions{}, wasmOptions, baseapp.SetChainID(testutil.TestChainID),
	)

	validatorPV := sdkmock.NewPV()
	validatorPubKey, err := validatorPV.GetPubKey()
	require.NoError(t, err)
	validatorSet := tmtypes.NewValidatorSet([]*tmtypes.Validator{tmtypes.NewValidator(validatorPubKey, 1)})
	privateKey := &ethsecp256k1.PrivKey{Key: crypto.FromECDSA(key)}
	creator := sdk.AccAddress(crypto.PubkeyToAddress(key.PublicKey).Bytes())
	account := authtypes.NewBaseAccount(creator, privateKey.PubKey(), 0, 0)
	amount, ok := math.NewIntFromString("100000000000000000000")
	require.True(t, ok)
	balance := banktypes.Balance{
		Address: creator.String(),
		Coins:   sdk.NewCoins(sdk.NewCoin(xplatypes.DefaultDenom, amount)),
	}
	genesis, err := simtestutil.GenesisStateWithValSet(
		app.AppCodec(), app.DefaultGenesis(), validatorSet,
		[]authtypes.GenesisAccount{account}, balance,
	)
	require.NoError(t, err)

	var bankGenesis banktypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesis[banktypes.ModuleName], &bankGenesis)
	bankGenesis.DenomMetadata = append(bankGenesis.DenomMetadata, banktypes.Metadata{
		Base: xplatypes.DefaultDenom, Display: "xpla", Name: "XPLA", Symbol: "XPLA",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: xplatypes.DefaultDenom, Exponent: 0},
			{Denom: "xpla", Exponent: uint32(vmtypes.EighteenDecimals)},
		},
	})
	genesis[banktypes.ModuleName] = app.AppCodec().MustMarshalJSON(&bankGenesis)
	var evmGenesis vmtypes.GenesisState
	app.AppCodec().MustUnmarshalJSON(genesis[vmtypes.ModuleName], &evmGenesis)
	evmGenesis.Params.EvmDenom = xplatypes.DefaultDenom
	evmGenesis.Params.ExtendedDenomOptions = &vmtypes.ExtendedDenomOptions{ExtendedDenom: xplatypes.DefaultDenom}
	genesis[vmtypes.ModuleName] = app.AppCodec().MustMarshalJSON(&evmGenesis)

	genesisState, err := json.Marshal(genesis)
	require.NoError(t, err)
	_, err = app.InitChain(&abci.RequestInitChain{
		ChainId:       testutil.TestChainID,
		AppStateBytes: genesisState,
		ConsensusParams: &tmproto.ConsensusParams{
			Block: &tmproto.BlockParams{MaxBytes: 20_000_000, MaxGas: 100_000_000},
			Validator: &tmproto.ValidatorParams{
				PubKeyTypes: []string{tmtypes.ABCIPubKeyTypeEd25519},
			},
		},
	})
	require.NoError(t, err)
	_, err = app.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: 1, Time: time.Unix(1, 0).UTC(), ProposerAddress: validatorPubKey.Address(),
	})
	require.NoError(t, err)
	_, err = app.Commit()
	require.NoError(t, err)
	return app, append([]byte(nil), validatorPubKey.Address()...)
}

func storeAndInstantiateRoutedContract(t *testing.T, app *xplaapp.XplaApp, proposer []byte, key *ecdsa.PrivateKey) sdk.AccAddress {
	t.Helper()
	wasmCode, err := os.ReadFile("../../solidity/suites/misc/any_dispatch.wasm")
	require.NoError(t, err)
	creator := sdk.AccAddress(crypto.PubkeyToAddress(key.PublicKey).Bytes())
	input := testutil.TestInput{App: app, Ctx: appContext(app, proposer)}
	result := commitCosmosMsg(t, &input, key, &wasmtypes.MsgStoreCode{
		Sender: creator.String(), WASMByteCode: wasmCode,
	})
	require.Zero(t, result.Code, result.Log)
	codeID := eventUint64(t, result.Events, "code_id")
	result = commitCosmosMsg(t, &input, key, &wasmtypes.MsgInstantiateContract{
		Sender: creator.String(), CodeID: codeID, Label: "genesis restart alias", Msg: []byte(`{}`),
	})
	require.Zero(t, result.Code, result.Log)
	return requireNewContract(t, result.Events, nil)
}

func requireRegisteredAppAlias(t *testing.T, app *xplaapp.XplaApp, contract sdk.AccAddress) {
	t.Helper()
	ctx := appContext(app, nil)
	resolved, found, err := app.WasmKeeper.ResolveWasmAlias(ctx, contract[len(contract)-20:])
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, contract, resolved)
	wasmAliases := storetypes.KVStorePrefixIterator(ctx.KVStore(app.GetKey(wasmtypes.StoreKey)), collections.NewPrefix("wasmAlias").Bytes())
	defer func() { require.NoError(t, wasmAliases.Close()) }()
	require.True(t, wasmAliases.Valid(), "alias must persist in the Wasm store")
	legacyAliases := storetypes.KVStorePrefixIterator(ctx.KVStore(app.GetKey(authtypes.StoreKey)), collections.NewPrefix("sliceAddress").Bytes())
	defer func() { require.NoError(t, legacyAliases.Close()) }()
	require.False(t, legacyAliases.Valid(), "alias lifecycle must not write the legacy auth registry")
}

func appContext(app *xplaapp.XplaApp, proposer []byte) sdk.Context {
	height := app.LastBlockHeight()
	return app.NewUncachedContext(false, tmproto.Header{
		ChainID: testutil.TestChainID, Height: height, Time: time.Unix(height, 0).UTC(), ProposerAddress: proposer,
	})
}

func eventUint64(t *testing.T, events []abci.Event, key string) uint64 {
	t.Helper()
	for _, event := range events {
		for _, attribute := range event.Attributes {
			if attribute.Key == key {
				value, err := strconv.ParseUint(attribute.Value, 10, 64)
				require.NoError(t, err)
				return value
			}
		}
	}
	t.Fatalf("event attribute %q not found", key)
	return 0
}

func persistentWasmKey(t *testing.T) *ecdsa.PrivateKey {
	t.Helper()
	key, err := crypto.HexToECDSA("997ec3668a447070a52bbe46563d4d95fd8dfaec992d4be659a424e71c720a45")
	require.NoError(t, err)
	return key
}

func captureWasmEngine(t *testing.T) (wasmkeeper.Option, func()) {
	t.Helper()
	var engine wasmtypes.WasmEngine
	option := wasmkeeper.WithWasmEngineDecorator(func(actual wasmtypes.WasmEngine) wasmtypes.WasmEngine {
		engine = actual
		return actual
	})
	cleanup := func() {
		if engine != nil {
			engine.Cleanup()
			engine = nil
		}
	}
	t.Cleanup(cleanup)
	return option, cleanup
}
