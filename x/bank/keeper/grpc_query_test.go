package keeper

import (
	"bytes"
	"context"
	"math/big"
	"testing"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

func TestGRPCQueriesRejectMalformedReservedDenom(t *testing.T) {
	address := sdk.AccAddress(bytes.Repeat([]byte{1}, 20)).String()
	keeper := Keeper{ak: moduleAccountKeeper{}}

	tests := []struct {
		name  string
		query func() error
	}{
		{
			name: "balance",
			query: func() error {
				_, err := keeper.Balance(context.Background(), &banktypes.QueryBalanceRequest{
					Address: address,
					Denom:   "xerc20:invalid",
				})
				return err
			},
		},
		{
			name: "spendable balance",
			query: func() error {
				_, err := keeper.SpendableBalanceByDenom(context.Background(), &banktypes.QuerySpendableBalanceByDenomRequest{
					Address: address,
					Denom:   "xcw20:invalid",
				})
				return err
			},
		},
		{
			name: "supply",
			query: func() error {
				_, err := keeper.SupplyOf(context.Background(), &banktypes.QuerySupplyOfRequest{
					Denom: "xcw20:invalid",
				})
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, codes.InvalidArgument, status.Code(tt.query()))
		})
	}
}

func TestGRPCBalanceReturnsZeroForValidUnrecordedAccount(t *testing.T) {
	returnData, err := ABI.Methods[xbanktypes.GetErc20Method(xbanktypes.BalanceOf)].Outputs.Pack(big.NewInt(0))
	require.NoError(t, err)

	executor := &recordingERC20EVMExecutor{
		callResponse: &evmtypes.MsgEthereumTxResponse{Ret: returnData},
	}
	accountKeeper := moduleAccountKeeper{
		moduleAccount: authtypes.NewEmptyModuleAccount(banktypes.ModuleName),
	}
	keeper := Keeper{bek: NewBaseErc20Keeper(accountKeeper, executor)}
	accountAddress := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	ctx := sdk.Context{}.
		WithContext(context.Background()).
		WithEventManager(sdk.NewEventManager()).
		WithGasMeter(storetypes.NewGasMeter(100_000))

	response, err := keeper.Balance(ctx, &banktypes.QueryBalanceRequest{
		Address: accountAddress.String(),
		Denom:   "xerc20:0xA2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546",
	})

	require.NoError(t, err)
	require.NotNil(t, response.Balance)
	require.True(t, response.Balance.Amount.IsZero())
	require.Equal(t, 1, executor.callCalls)
}
