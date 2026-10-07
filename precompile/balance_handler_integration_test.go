package precompile_test

import (
	"bytes"
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	cmn "github.com/cosmos/evm/precompiles/common"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	pbank "github.com/xpladev/xpla/precompile/bank"
	pics20 "github.com/xpladev/xpla/precompile/ics20"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
	"github.com/xpladev/xpla/tests/integration/testutil"
	xplatypes "github.com/xpladev/xpla/types"
)

const precompileTestGas = uint64(10_000_000)

func TestProductionBalanceHandlersIgnoreNonEVMAddressEvents(t *testing.T) {
	input := testutil.CreateTestInput(t)
	alias := common.HexToAddress("0x1111111111111111111111111111111111111111")
	aliasAddr := sdk.AccAddress(alias.Bytes())
	longAddr := sdk.AccAddress(append(bytes.Repeat([]byte{0x42}, 12), alias.Bytes()...))
	coins := sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 10))
	reward := sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 7))
	require.NoError(t, input.InitAccountWithCoins(aliasAddr, coins))
	require.False(t, input.BankKeeper.BlockedAddr(longAddr),
		"native bank operations must continue to support non-EVM addresses")

	tests := []struct {
		name    string
		address common.Address
	}{
		{name: "staking", address: common.HexToAddress(evmtypes.StakingPrecompileAddress)},
		{name: "distribution", address: common.HexToAddress(evmtypes.DistributionPrecompileAddress)},
		{name: "ics20", address: common.HexToAddress(evmtypes.ICS20PrecompileAddress)},
		{name: "gov", address: common.HexToAddress(evmtypes.GovPrecompileAddress)},
		{name: "slashing", address: common.HexToAddress(evmtypes.SlashingPrecompileAddress)},
		{name: "bank", address: pbank.Address},
		{name: "wasm", address: pwasm.Address},
		{name: "wasm delegate", address: pwasm.DelegatecallAddress},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			factory := productionBalanceHandlerFactory(t, &input, tc.address)
			ctx := input.Ctx.WithEventManager(sdk.NewEventManager())
			db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
			require.Equal(t, uint64(10), db.GetBalance(alias).Uint64())

			handler := factory.NewBalanceHandler()
			handler.BeforeBalanceChange(ctx)
			ctx.EventManager().EmitEvent(banktypes.NewCoinReceivedEvent(longAddr, reward))
			require.NoError(t, handler.AfterBalanceChange(ctx, db))

			require.Equal(t, uint64(10), db.GetBalance(alias).Uint64(),
				"a non-EVM receiver must not be projected onto its 20-byte suffix")
		})
	}
}

func TestDistributionPrecompileRewardToLongAddressDoesNotCreditSuffixEOA(t *testing.T) {
	fixture := newRewardedDelegationFixture(t)
	ctx := fixture.input.Ctx.WithGasMeter(storetypes.NewGasMeter(precompileTestGas)).WithEventManager(sdk.NewEventManager())

	aliasBefore := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, xplatypes.DefaultDenom)
	withdrawBefore := fixture.input.BankKeeper.GetBalance(ctx, fixture.withdrawAddr, xplatypes.DefaultDenom)
	supplyBefore := fixture.input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom)

	callData, err := distprecompile.ABI.Pack(
		distprecompile.WithdrawDelegatorRewardMethod,
		fixture.delegator,
		fixture.validatorAddr.String(),
	)
	require.NoError(t, err)

	db := runProductionPrecompile(t, &fixture.input, ctx,
		common.HexToAddress(evmtypes.DistributionPrecompileAddress), fixture.delegator, callData)

	aliasAfter := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, xplatypes.DefaultDenom)
	withdrawAfter := fixture.input.BankKeeper.GetBalance(ctx, fixture.withdrawAddr, xplatypes.DefaultDenom)
	require.Equal(t, aliasBefore, aliasAfter)
	require.True(t, withdrawAfter.Amount.GT(withdrawBefore.Amount), "the exact 32-byte withdraw address must receive rewards")
	require.Equal(t, supplyBefore, fixture.input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom))
	require.Equal(t, aliasBefore.Amount.Uint64(), db.GetBalance(fixture.delegator).Uint64())
}

func TestStakingPrecompileRewardHookToLongAddressDoesNotCreditSuffixEOA(t *testing.T) {
	fixture := newRewardedDelegationFixture(t)
	ctx := fixture.input.Ctx.WithGasMeter(storetypes.NewGasMeter(precompileTestGas)).WithEventManager(sdk.NewEventManager())

	aliasBefore := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, xplatypes.DefaultDenom)
	withdrawBefore := fixture.input.BankKeeper.GetBalance(ctx, fixture.withdrawAddr, xplatypes.DefaultDenom)
	stakeBefore := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, sdk.DefaultBondDenom)
	supplyBefore := fixture.input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom)

	callData, err := stakingprecompile.ABI.Pack(
		stakingprecompile.DelegateMethod,
		fixture.delegator,
		fixture.validatorAddr.String(),
		fixture.extraDelegation.BigInt(),
	)
	require.NoError(t, err)

	db := runProductionPrecompile(t, &fixture.input, ctx,
		common.HexToAddress(evmtypes.StakingPrecompileAddress), fixture.delegator, callData)

	aliasAfter := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, xplatypes.DefaultDenom)
	withdrawAfter := fixture.input.BankKeeper.GetBalance(ctx, fixture.withdrawAddr, xplatypes.DefaultDenom)
	stakeAfter := fixture.input.BankKeeper.GetBalance(ctx, fixture.delegatorAddr, sdk.DefaultBondDenom)
	require.Equal(t, aliasBefore, aliasAfter)
	require.True(t, withdrawAfter.Amount.GT(withdrawBefore.Amount),
		"the distribution hook must pay rewards to the exact 32-byte withdraw address")
	require.True(t, stakeAfter.Amount.Equal(stakeBefore.Amount.Sub(fixture.extraDelegation)),
		"the production staking action must execute")
	require.Equal(t, supplyBefore, fixture.input.BankKeeper.GetSupply(ctx, xplatypes.DefaultDenom))
	require.Equal(t, aliasBefore.Amount.Uint64(), db.GetBalance(fixture.delegator).Uint64())
}

type rewardedDelegationFixture struct {
	input           testutil.TestInput
	delegator       common.Address
	delegatorAddr   sdk.AccAddress
	withdrawAddr    sdk.AccAddress
	validatorAddr   sdk.ValAddress
	extraDelegation sdkmath.Int
}

func newRewardedDelegationFixture(t *testing.T) rewardedDelegationFixture {
	t.Helper()

	input := testutil.CreateTestInput(t)
	delegator := common.HexToAddress("0x1111111111111111111111111111111111111111")
	delegatorAddr := sdk.AccAddress(delegator.Bytes())
	withdrawAddr := sdk.AccAddress(append(bytes.Repeat([]byte{0x42}, 12), delegator.Bytes()...))
	validatorAddr := sdk.ValAddress(testutil.Pks[0].Address())
	validatorAccount := sdk.AccAddress(testutil.Pks[0].Address())
	selfBond := input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 1)
	delegation := input.StakingKeeper.TokensFromConsensusPower(input.Ctx, 1)
	extraDelegation := sdkmath.OneInt()

	require.NoError(t, input.InitAccountWithCoins(validatorAccount,
		sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, selfBond))))
	require.NoError(t, input.InitAccountWithCoins(delegatorAddr, sdk.NewCoins(
		sdk.NewInt64Coin(xplatypes.DefaultDenom, 10),
		sdk.NewCoin(sdk.DefaultBondDenom, delegation.Add(extraDelegation)),
	)))
	require.NoError(t, input.InitAccountWithCoins(withdrawAddr,
		sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 30))))

	input.StakingHandler.CreateValidator(validatorAddr, testutil.Pks[0], selfBond, true)
	input.StakingHandler.Delegate(delegatorAddr, validatorAddr, delegation)
	input.Ctx = input.Ctx.WithBlockHeight(input.Ctx.BlockHeight() + 1)

	reward := sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 100))
	require.NoError(t, input.BankKeeper.MintCoins(input.Ctx, minttypes.ModuleName, reward))
	require.NoError(t, input.BankKeeper.SendCoinsFromModuleToModule(
		input.Ctx, minttypes.ModuleName, distrtypes.ModuleName, reward))

	validator, err := input.StakingKeeper.GetValidator(input.Ctx, validatorAddr)
	require.NoError(t, err)
	require.NoError(t, input.DistrKeeper.AllocateTokensToValidator(
		input.Ctx, validator, sdk.NewDecCoins(sdk.NewDecCoinFromCoin(reward[0]))))
	require.NoError(t, input.DistrKeeper.SetWithdrawAddr(input.Ctx, delegatorAddr, withdrawAddr))
	input.Ctx = input.Ctx.WithEventManager(sdk.NewEventManager())

	return rewardedDelegationFixture{
		input:           input,
		delegator:       delegator,
		delegatorAddr:   delegatorAddr,
		withdrawAddr:    withdrawAddr,
		validatorAddr:   validatorAddr,
		extraDelegation: extraDelegation,
	}
}

func runProductionPrecompile(
	t *testing.T,
	input *testutil.TestInput,
	ctx sdk.Context,
	address common.Address,
	caller common.Address,
	callData []byte,
) *statedb.StateDB {
	t.Helper()

	precompile := productionPrecompile(t, input, address)
	db := statedb.New(ctx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
	evm := vm.NewEVM(vm.BlockContext{BlockNumber: big.NewInt(ctx.BlockHeight())}, db, evmtypes.GetEthChainConfig(), vm.Config{})
	evm.Origin = caller
	contract := vm.NewContract(caller, address, uint256.NewInt(0), precompileTestGas, nil)
	contract.Input = callData
	_, err := precompile.Run(evm, contract, false)
	require.NoError(t, err)
	require.NoError(t, db.Commit())
	return db
}

func productionBalanceHandlerFactory(
	t *testing.T,
	input *testutil.TestInput,
	address common.Address,
) *cmn.BalanceHandlerFactory {
	t.Helper()

	switch precompile := productionPrecompile(t, input, address).(type) {
	case *stakingprecompile.Precompile:
		return precompile.BalanceHandlerFactory
	case *distprecompile.Precompile:
		return precompile.BalanceHandlerFactory
	case pics20.Precompile:
		return precompile.BalanceHandlerFactory
	case *pics20.Precompile:
		return precompile.BalanceHandlerFactory
	case *govprecompile.Precompile:
		return precompile.BalanceHandlerFactory
	case *slashingprecompile.Precompile:
		return precompile.BalanceHandlerFactory
	case pbank.PrecompiledBank:
		return precompile.BalanceHandlerFactory
	case *pwasm.PrecompiledWasm:
		return precompile.BalanceHandlerFactory
	case pwasm.DelegatePrecompile:
		return precompile.BalanceHandlerFactory
	default:
		t.Fatalf("precompile %s does not expose a balance handler factory", address)
		return nil
	}
}

func productionPrecompile(
	t *testing.T,
	input *testutil.TestInput,
	address common.Address,
) vm.PrecompiledContract {
	t.Helper()

	params := input.App.EvmKeeper.GetParams(input.Ctx)
	params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, address.String())
	precompile, found, err := input.App.EvmKeeper.GetStaticPrecompileInstance(&params, address)
	require.NoError(t, err)
	require.True(t, found)
	require.NotNil(t, precompile)
	return precompile
}
