package testutil

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtestutil "github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/cosmos/evm/x/feemarket"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	ratelimittypes "github.com/cosmos/ibc-apps/modules/rate-limiting/v10/types"
	transfertypes "github.com/cosmos/ibc-go/v10/modules/apps/transfer/types"
	ibc "github.com/cosmos/ibc-go/v10/modules/core"
	clienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v10/modules/core/03-connection/types"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	commitmenttypes "github.com/cosmos/ibc-go/v10/modules/core/23-commitment/types"
	"github.com/cosmos/ibc-go/v10/modules/core/exported"
	ibctypes "github.com/cosmos/ibc-go/v10/modules/core/types"
	tendermint "github.com/cosmos/ibc-go/v10/modules/light-clients/07-tendermint"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	xplaApp "github.com/xpladev/xpla/app"
	xplatypes "github.com/xpladev/xpla/types"
	authkeeper "github.com/xpladev/xpla/x/auth/keeper"
	bankkeeper "github.com/xpladev/xpla/x/bank/keeper"
	dynamicdeflationtypes "github.com/xpladev/xpla/x/dynamicdeflation/types"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
	volunteerkeeper "github.com/xpladev/xpla/x/volunteer/keeper"
)

const (
	TestChainID = "integration_37-1"

	ValidatorCount = 10

	ValidatorSettlementIndex = ValidatorCount
	ReserveIndex             = ValidatorCount + 1
	TempIndex                = ValidatorCount + 2
	ProposerIndex            = ValidatorCount + 3

	TotalCount = ProposerIndex + 1
)

var (
	Pks = simtestutil.CreateTestPubKeys(TotalCount)
)

// TestInput nolint
type TestInput struct {
	App             *xplaApp.XplaApp
	Ctx             sdk.Context
	Cdc             *codec.LegacyAmino
	AccountKeeper   authkeeper.AccountKeeper
	BankKeeper      bankkeeper.Keeper
	RewardKeeper    rewardkeeper.Keeper
	StakingKeeper   *stakingkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	DistrKeeper     distrkeeper.Keeper
	VolunteerKeeper volunteerkeeper.Keeper

	StakingHandler *stakingtestutil.Helper
}

// CreateTestInput initializes keeper, EVM and IBC state with a proposer record.
// Signed transactions can start at block 1 without committing an empty block.
func CreateTestInput(t *testing.T) TestInput {
	t.Helper()
	app := xplaApp.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		t.TempDir(),
		xplaApp.EmptyAppOptions{},
		xplaApp.EmptyWasmOptions,
		baseapp.SetChainID(TestChainID),
	)
	t.Cleanup(func() { require.NoError(t, app.Close()) })

	ctx := app.BaseApp.NewUncachedContext(false, tmproto.Header{ChainID: TestChainID, Time: time.Unix(0, 0).UTC()})
	keepers := app.AppKeepers
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = sdk.DefaultBondDenom
	keepers.StakingKeeper.SetParams(ctx, stakingParams)

	// EVM coinbase lookup needs a validator record, independent of test validators.
	proposer := Pks[ProposerIndex]
	validator, err := stakingtypes.NewValidator(sdk.ValAddress(proposer.Address()).String(), proposer, stakingtypes.Description{})
	require.NoError(t, err)
	require.NoError(t, keepers.StakingKeeper.SetValidator(ctx, validator))
	require.NoError(t, keepers.StakingKeeper.SetValidatorByConsAddr(ctx, validator))
	ctx = ctx.WithProposer(sdk.ConsAddress(proposer.Address()))

	keepers.SlashingKeeper.SetParams(ctx, slashingtypes.DefaultParams())

	keepers.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = sdkmath.LegacyZeroDec()
	keepers.DistrKeeper.Params.Set(ctx, distrParams)

	keepers.MintKeeper.InitGenesis(ctx, keepers.AccountKeeper, minttypes.DefaultGenesisState())

	rewardParams := rewardtypes.Params{
		FeePoolRate:             sdkmath.LegacyNewDecWithPrec(20, 2),
		CommunityPoolRate:       sdkmath.LegacyNewDecWithPrec(79, 2),
		ReserveRate:             sdkmath.LegacyNewDecWithPrec(1, 2),
		ReserveAccount:          sdk.AccAddress(Pks[ReserveIndex].Address()).String(),
		RewardDistributeAccount: sdk.AccAddress(Pks[ValidatorSettlementIndex].Address()).String(),
	}
	keepers.RewardKeeper.SetParams(ctx, rewardParams)

	// State required by transaction ante-handlers and the real block lifecycle.
	keepers.AccountKeeper.InitGenesis(ctx, *authtypes.DefaultGenesisState())
	require.NoError(t, keepers.DynamicDeflationKeeper.InitGenesis(ctx, dynamicdeflationtypes.DefaultGenesisState()))
	require.NoError(t, app.StoreConsensusParams(ctx, tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{MaxBytes: 20_000_000, MaxGas: 100_000_000},
	}))

	bankGenesis := banktypes.DefaultGenesisState()
	bankGenesis.DenomMetadata = []banktypes.Metadata{{
		Base: xplatypes.DefaultDenom, Display: "xpla", Name: "XPLA", Symbol: "XPLA",
		DenomUnits: []*banktypes.DenomUnit{
			{Denom: xplatypes.DefaultDenom, Exponent: 0},
			{Denom: "xpla", Exponent: uint32(vmtypes.EighteenDecimals)},
		},
	}}
	keepers.BankKeeper.InitGenesis(ctx, bankGenesis)
	feemarket.InitGenesis(ctx, keepers.FeeMarketKeeper, *feemarkettypes.DefaultGenesisState())
	evmParams := vmtypes.DefaultParams()
	evmParams.EvmDenom = xplatypes.DefaultDenom
	evmParams.ExtendedDenomOptions = &vmtypes.ExtendedDenomOptions{ExtendedDenom: xplatypes.DefaultDenom}
	evmParams.ActiveStaticPrecompiles = append(evmParams.ActiveStaticPrecompiles,
		common.HexToAddress("0x0802").Hex(), common.HexToAddress("0x1000000000000000000000000000000000000001").Hex())
	require.NoError(t, keepers.EvmKeeper.SetParams(ctx, evmParams))
	// The app's first PreBlock initializes process-wide EVM configuration.
	require.NoError(t, keepers.EvmKeeper.InitEvmCoinInfo(ctx))

	ibc.InitGenesis(ctx, *keepers.IBCKeeper, ibctypes.DefaultGenesisState())
	keepers.TransferKeeper.InitGenesis(ctx, *transfertypes.DefaultGenesisState())
	keepers.RatelimitKeeper.InitGenesis(ctx, *ratelimittypes.DefaultGenesis())

	// Seed a real Tendermint client and open channel for local send/rollback tests.
	// The multichain suite covers the actual handshake, proofs and relayer.
	height := clienttypes.NewHeight(1, 1)
	client := tendermint.NewClientState("ics20-destination-1", tendermint.DefaultTrustLevel,
		time.Hour, 2*time.Hour, time.Minute, height, commitmenttypes.GetSDKSpecs(), []string{"upgrade", "upgradedIBCState"})
	consensus := tendermint.NewConsensusState(time.Unix(1, 0).UTC(), commitmenttypes.NewMerkleRoot(bytes.Repeat([]byte{1}, 32)), bytes.Repeat([]byte{2}, 32))
	clientID, err := app.IBCKeeper.ClientKeeper.CreateClient(ctx, exported.Tendermint,
		app.AppCodec().MustMarshal(client), app.AppCodec().MustMarshal(consensus))
	require.NoError(t, err)
	app.IBCKeeper.ConnectionKeeper.SetConnection(ctx, "connection-0", connectiontypes.NewConnectionEnd(
		connectiontypes.OPEN, clientID, connectiontypes.NewCounterparty("07-tendermint-0", "connection-0", commitmenttypes.NewMerklePrefix([]byte("ibc"))), connectiontypes.GetCompatibleVersions(), 0))
	app.IBCKeeper.ChannelKeeper.SetChannel(ctx, "transfer", "channel-0", channeltypes.NewChannel(
		channeltypes.OPEN, channeltypes.UNORDERED, channeltypes.NewCounterparty("transfer", "channel-0"), []string{"connection-0"}, transfertypes.V1))
	app.IBCKeeper.ChannelKeeper.SetNextSequenceSend(ctx, "transfer", "channel-0", 1)

	sh := stakingtestutil.NewHelper(t, ctx, app.AppKeepers.StakingKeeper.Keeper)
	app.ModuleBasics.RegisterInterfaces(app.InterfaceRegistry())

	return TestInput{
		App:             app,
		Ctx:             ctx,
		Cdc:             app.LegacyAmino(),
		AccountKeeper:   keepers.AccountKeeper,
		BankKeeper:      keepers.BankKeeper,
		RewardKeeper:    keepers.RewardKeeper,
		StakingKeeper:   keepers.StakingKeeper,
		SlashingKeeper:  keepers.SlashingKeeper,
		DistrKeeper:     keepers.DistrKeeper,
		VolunteerKeeper: keepers.VolunteerKeeper,
		StakingHandler:  sh,
	}
}

func (ti *TestInput) InitAccountWithCoins(addr sdk.AccAddress, coins sdk.Coins) error {
	err := ti.BankKeeper.MintCoins(ti.Ctx, minttypes.ModuleName, coins)
	if err != nil {
		return err
	}

	err = ti.BankKeeper.SendCoinsFromModuleToAccount(ti.Ctx, minttypes.ModuleName, addr, coins)
	if err != nil {
		return err
	}

	return nil
}

// CommitTransaction executes one transaction in the next block and commits its state.
// Test block timestamps use the block height as Unix seconds.
func (ti *TestInput) CommitTransaction(t *testing.T, txBytes []byte) *abci.ExecTxResult {
	t.Helper()
	height := ti.App.LastBlockHeight() + 1
	block, err := ti.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Txs: [][]byte{txBytes}, Height: height, Time: time.Unix(height, 0).UTC(),
		ProposerAddress: ti.Ctx.BlockHeader().ProposerAddress,
	})
	require.NoError(t, err)
	_, err = ti.App.Commit()
	require.NoError(t, err)
	require.Len(t, block.TxResults, 1)
	result := block.TxResults[0]
	require.Zero(t, result.Code, result.Log)
	return result
}
