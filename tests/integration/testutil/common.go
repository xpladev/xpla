package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	simparams "cosmossdk.io/simapp/params"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/bank"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/cosmos-sdk/x/mint"
	mintkeeper "github.com/cosmos/cosmos-sdk/x/mint/keeper"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	params "github.com/cosmos/cosmos-sdk/x/params"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingtestutil "github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	etherminttypes "github.com/xpladev/ethermint/types"

	"github.com/xpladev/xpla/x/reward"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
	"github.com/xpladev/xpla/x/volunteer"
	volunteerkeeper "github.com/xpladev/xpla/x/volunteer/keeper"
	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"
)

const (
	ValidatorCount = 10

	ValidatorSettlementIndex = ValidatorCount
	ReserveIndex             = ValidatorCount + 1
	TempIndex                = ValidatorCount + 2

	TotalCount = TempIndex + 1
)

var (
	Pks = simtestutil.CreateTestPubKeys(TotalCount)
)

// ModuleBasics nolint
var ModuleBasics = module.NewBasicManager(
	auth.AppModuleBasic{},
	bank.AppModuleBasic{},
	distr.AppModuleBasic{},
	staking.AppModuleBasic{},
	slashing.AppModuleBasic{},
	mint.AppModuleBasic{},
	params.AppModuleBasic{},
	reward.AppModuleBasic{},
	volunteer.AppModuleBasic{},
)

// MakeEncodingConfig nolint
func MakeEncodingConfig(_ *testing.T) simparams.EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	etherminttypes.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)

	return simparams.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Codec:             marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// TestInput nolint
type TestInput struct {
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

// CreateTestInput nolint
func CreateTestInput(t *testing.T) TestInput {
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyBank := sdk.NewKVStoreKey(banktypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	tKeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)
	keyReward := sdk.NewKVStoreKey(rewardtypes.StoreKey)
	keyStaking := sdk.NewKVStoreKey(stakingtypes.StoreKey)
	keySlahsing := sdk.NewKVStoreKey(slashingtypes.StoreKey)
	keyDistr := sdk.NewKVStoreKey(distrtypes.StoreKey)
	keyMint := sdk.NewKVStoreKey(minttypes.StoreKey)
	keyVolunteer := sdk.NewKVStoreKey(volunteertypes.StoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ctx := sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Codec, encodingConfig.Amino

	ms.MountStoreWithDB(keyAcc, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, storetypes.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyReward, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySlahsing, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistr, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyMint, storetypes.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyVolunteer, storetypes.StoreTypeIAVL, db)

	require.NoError(t, ms.LoadLatestVersion())

	blackListAddrs := map[string]bool{
		authtypes.FeeCollectorName:     true,
		stakingtypes.NotBondedPoolName: true,
		stakingtypes.BondedPoolName:    true,
		distrtypes.ModuleName:          true,
	}

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName:     nil,
		minttypes.ModuleName:           {authtypes.Minter},
		stakingtypes.NotBondedPoolName: {authtypes.Burner, authtypes.Staking},
		stakingtypes.BondedPoolName:    {authtypes.Burner, authtypes.Staking},
		distrtypes.ModuleName:          nil,
		rewardtypes.ModuleName:         nil,
	}

	govModAddress := authtypes.NewModuleAddress(govtypes.ModuleName).String()

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accountKeeper := authkeeper.NewAccountKeeper(appCodec, keyAcc, etherminttypes.ProtoAccount, maccPerms, sdk.GetConfig().GetBech32AccountAddrPrefix(), govModAddress)
	bankKeeper := bankkeeper.NewBaseKeeper(appCodec, keyBank, accountKeeper, blackListAddrs, govModAddress)

	var volunteerKeeper volunteerkeeper.Keeper
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keyStaking,
		accountKeeper,
		bankKeeper,
		govModAddress,
		&volunteerKeeper,
	)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = sdk.DefaultBondDenom
	stakingKeeper.SetParams(ctx, stakingParams)

	mintKeeper := mintkeeper.NewKeeper(appCodec, keyMint, stakingKeeper, accountKeeper, bankKeeper, authtypes.FeeCollectorName, govModAddress)

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		keyDistr,
		accountKeeper, bankKeeper, stakingKeeper,
		authtypes.FeeCollectorName, govModAddress)

	slashingKeeper := slashingkeeper.NewKeeper(appCodec, legacyAmino, keySlahsing, stakingKeeper, govModAddress)
	slashingKeeper.SetParams(ctx, slashingtypes.DefaultParams())

	volunteerKeeper = volunteerkeeper.NewKeeper(keyVolunteer, appCodec, stakingKeeper, distrKeeper)

	distrKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = sdk.ZeroDec()
	distrKeeper.SetParams(ctx, distrParams)
	stakingKeeper.SetHooks(stakingtypes.NewMultiStakingHooks(distrKeeper.Hooks(), slashingKeeper.Hooks()))
	mintKeeper.SetParams(ctx, minttypes.DefaultParams())

	feeCollectorAcc := authtypes.NewEmptyModuleAccount(authtypes.FeeCollectorName)
	notBondedPool := authtypes.NewEmptyModuleAccount(stakingtypes.NotBondedPoolName, authtypes.Burner, authtypes.Staking)
	bondPool := authtypes.NewEmptyModuleAccount(stakingtypes.BondedPoolName, authtypes.Burner, authtypes.Staking)
	distrAcc := authtypes.NewEmptyModuleAccount(distrtypes.ModuleName)
	rewardAcc := authtypes.NewEmptyModuleAccount(rewardtypes.ModuleName)

	accountKeeper.SetModuleAccount(ctx, feeCollectorAcc)
	accountKeeper.SetModuleAccount(ctx, bondPool)
	accountKeeper.SetModuleAccount(ctx, notBondedPool)
	accountKeeper.SetModuleAccount(ctx, distrAcc)
	accountKeeper.SetModuleAccount(ctx, rewardAcc)

	keeper := rewardkeeper.NewKeeper(
		appCodec,
		keyReward,
		paramsKeeper.Subspace(rewardtypes.ModuleName),
		accountKeeper,
		bankKeeper,
		stakingKeeper,
		distrKeeper,
		mintKeeper,
	)

	defaults := rewardtypes.Params{
		FeePoolRate:             sdk.NewDecWithPrec(20, 2),
		CommunityPoolRate:       sdk.NewDecWithPrec(79, 2),
		ReserveRate:             sdk.NewDecWithPrec(1, 2),
		ReserveAccount:          sdk.AccAddress(Pks[ReserveIndex].Address()).String(),
		RewardDistributeAccount: sdk.AccAddress(Pks[ValidatorSettlementIndex].Address()).String(),
	}
	keeper.SetParams(ctx, defaults)

	sh := stakingtestutil.NewHelper(t, ctx, stakingKeeper.Keeper)

	return TestInput{ctx, legacyAmino, accountKeeper, bankKeeper, keeper, stakingKeeper, slashingKeeper, distrKeeper, volunteerKeeper, sh}
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
