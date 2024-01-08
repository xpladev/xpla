package testutil

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/simapp"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
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
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
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
	Pks = simapp.CreateTestPubKeys(TotalCount)
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
)

// MakeEncodingConfig nolint
func MakeEncodingConfig(_ *testing.T) simparams.EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	rewardtypes.RegisterLegacyAminoCodec(amino)
	rewardtypes.RegisterInterfaces(interfaceRegistry)
	return simparams.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
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
	StakingKeeper   stakingkeeper.Keeper
	SlashingKeeper  slashingkeeper.Keeper
	DistrKeeper     distrkeeper.Keeper
	VolunteerKeeper volunteerkeeper.Keeper

	StakingHandler sdk.Handler
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
	appCodec, legacyAmino := encodingConfig.Marshaler, encodingConfig.Amino

	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyBank, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, sdk.StoreTypeTransient, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyReward, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyStaking, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keySlahsing, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyDistr, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyMint, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyVolunteer, sdk.StoreTypeIAVL, db)

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

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accountKeeper := authkeeper.NewAccountKeeper(appCodec, keyAcc, paramsKeeper.Subspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, maccPerms)
	bankKeeper := bankkeeper.NewBaseKeeper(appCodec, keyBank, accountKeeper, paramsKeeper.Subspace(banktypes.ModuleName), blackListAddrs)

	var volunteerKeeper volunteerkeeper.Keeper
	stakingKeeper := stakingkeeper.NewKeeper(
		appCodec,
		keyStaking,
		accountKeeper,
		bankKeeper,
		paramsKeeper.Subspace(stakingtypes.ModuleName),
		&volunteerKeeper,
	)

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = sdk.DefaultBondDenom
	stakingKeeper.SetParams(ctx, stakingParams)

	mintKeeper := mintkeeper.NewKeeper(appCodec, keyMint, paramsKeeper.Subspace(minttypes.ModuleName), stakingKeeper, accountKeeper, bankKeeper, authtypes.FeeCollectorName)

	distrKeeper := distrkeeper.NewKeeper(
		appCodec,
		keyDistr, paramsKeeper.Subspace(distrtypes.ModuleName),
		accountKeeper, bankKeeper, stakingKeeper,
		authtypes.FeeCollectorName, blackListAddrs)

	slashingKeeper := slashingkeeper.NewKeeper(appCodec, keySlahsing, &stakingKeeper, paramsKeeper.Subspace(slashingtypes.ModuleName))
	slashingKeeper.SetParams(ctx, slashingtypes.DefaultParams())

	volunteerKeeper = volunteerkeeper.NewKeeper(keyVolunteer, appCodec, &stakingKeeper, distrKeeper)

	distrKeeper.SetFeePool(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = sdk.ZeroDec()
	distrParams.BaseProposerReward = sdk.ZeroDec()
	distrParams.BonusProposerReward = sdk.ZeroDec()
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

	sh := staking.NewHandler(stakingKeeper.Keeper)

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
