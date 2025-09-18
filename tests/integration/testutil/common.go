package testutil

import (
	"testing"
	"time"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"

	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtestutil "github.com/cosmos/cosmos-sdk/x/staking/testutil"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	xplaApp "github.com/xpladev/xpla/app"
	xplatypes "github.com/xpladev/xpla/types"
	authkeeper "github.com/xpladev/xpla/x/auth/keeper"
	bankkeeper "github.com/xpladev/xpla/x/bank/keeper"
	rewardkeeper "github.com/xpladev/xpla/x/reward/keeper"
	rewardtypes "github.com/xpladev/xpla/x/reward/types"
	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
	volunteerkeeper "github.com/xpladev/xpla/x/volunteer/keeper"
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
	app := xplaApp.NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		xplaApp.DefaultNodeHome,
		xplaApp.EmptyAppOptions{},
		xplaApp.EmptyWasmOptions,
		xplatypes.EvmAppOptions,
	)

	ctx := app.BaseApp.NewUncachedContext(true, tmproto.Header{Time: time.Now().UTC()})

	// Params setting
	keepers := app.AppKeepers

	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = sdk.DefaultBondDenom
	keepers.StakingKeeper.SetParams(ctx, stakingParams)

	keepers.SlashingKeeper.SetParams(ctx, slashingtypes.DefaultParams())

	keepers.DistrKeeper.FeePool.Set(ctx, distrtypes.InitialFeePool())
	distrParams := distrtypes.DefaultParams()
	distrParams.CommunityTax = sdkmath.LegacyZeroDec()
	keepers.DistrKeeper.Params.Set(ctx, distrParams)

	keepers.MintKeeper.Params.Set(ctx, minttypes.DefaultParams())

	rewardParams := rewardtypes.Params{
		FeePoolRate:             sdkmath.LegacyNewDecWithPrec(20, 2),
		CommunityPoolRate:       sdkmath.LegacyNewDecWithPrec(79, 2),
		ReserveRate:             sdkmath.LegacyNewDecWithPrec(1, 2),
		ReserveAccount:          sdk.AccAddress(Pks[ReserveIndex].Address()).String(),
		RewardDistributeAccount: sdk.AccAddress(Pks[ValidatorSettlementIndex].Address()).String(),
	}
	keepers.RewardKeeper.SetParams(ctx, rewardParams)

	sh := stakingtestutil.NewHelper(t, ctx, app.AppKeepers.StakingKeeper.Keeper)
	app.ModuleBasics.RegisterInterfaces(app.InterfaceRegistry())

	return TestInput{
		ctx,
		app.LegacyAmino(),
		app.AppKeepers.AccountKeeper,
		app.AppKeepers.BankKeeper,
		app.AppKeepers.RewardKeeper,
		app.AppKeepers.StakingKeeper,
		app.AppKeepers.SlashingKeeper,
		app.AppKeepers.DistrKeeper,
		app.AppKeepers.VolunteerKeeper,
		sh,
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
