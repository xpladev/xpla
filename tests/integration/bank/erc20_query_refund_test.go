package bank_test

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkbank "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/evm/x/vm/statedb"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/tests/integration/testutil"
	bankkeeper "github.com/xpladev/xpla/x/bank/keeper"
	banktypes "github.com/xpladev/xpla/x/bank/types"
)

func TestSharedERC20QueriesDoNotReuseOuterRefund(t *testing.T) {
	input := testutil.CreateTestInput(t)
	input.AccountKeeper.GetModuleAccount(input.EVMContext(), sdkbank.ModuleName)
	keeper := bankkeeper.NewErc20Keeper(input.AccountKeeper, input.App.EvmKeeper)
	token := common.HexToAddress("0x123400000000000000000000000000000000abcd")
	account := sdk.AccAddress(common.HexToAddress("0x5678").Bytes())

	for _, method := range []string{"balanceOf", "totalSupply"} {
		t.Run(method, func(t *testing.T) {
			costs := make(map[uint64][]uint64, 2)
			for _, refund := range []uint64{0, 1_000_000} {
				func() {
					baseCtx := input.EVMContext().WithGasMeter(storetypes.NewGasMeter(4_000_000))
					db := statedb.New(baseCtx, input.App.EvmKeeper, statedb.NewEmptyTxConfig())
					// Runtime bytecode returns uint256(42) without changing storage or
					// earning a gas refund. Any discount therefore came from the outer
					// StateDB rather than this view call.
					db.SetCode(token, common.FromHex("0x602a60005260206000f3"))
					actionCtx, endAction, err := db.BeginNativeAction()
					require.NoError(t, err)
					defer endAction()
					require.NoError(t, db.FlushToCacheCtx())
					// Reconciliation imports the live native balance (zero) inside each
					// query scope. Discard must restore this pending outer EVM value.
					db.SetBalance(token, uint256.NewInt(9), tracing.BalanceChangeUnspecified)
					db.AddRefund(refund)
					ctx := banktypes.WithEVMStateDB(actionCtx, db)

					for queryIndex := 0; queryIndex < 2; queryIndex++ {
						before := ctx.GasMeter().GasConsumed()
						var value string
						if method == "balanceOf" {
							amount, err := keeper.QueryBalanceOf(ctx, token, account)
							require.NoError(t, err)
							value = amount.String()
						} else {
							amount, err := keeper.QueryTotalSupply(ctx, token)
							require.NoError(t, err)
							value = amount.String()
						}

						require.Equal(t, "42", value)
						require.Equal(t, refund, db.GetRefund(), "outer refund changed")
						require.Equal(t, uint256.NewInt(9), db.GetBalance(token),
							"query scope leaked a reconciled native balance")
						used := ctx.GasMeter().GasConsumed() - before
						costs[refund] = append(costs[refund], used)
						t.Logf(
							"method=%s query=%d outer_refund=%d sdk_gas_delta=%d refund_after_query=%d",
							method,
							queryIndex+1,
							refund,
							used,
							db.GetRefund(),
						)
					}
				}()
			}

			require.Equal(t, costs[uint64(0)], costs[uint64(1_000_000)],
				"shared queries must not borrow a pre-existing outer refund")
		})
	}
}
