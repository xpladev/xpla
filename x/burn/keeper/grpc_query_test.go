package keeper

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/x/burn/types"
)

type burnQueryAccountKeeper struct {
	moduleAddress sdk.AccAddress
}

func (m burnQueryAccountKeeper) GetModuleAddress(string) sdk.AccAddress {
	return m.moduleAddress
}

func setupBurnQueryKeeper(t *testing.T) (Querier, sdk.Context) {
	t.Helper()

	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("burn_query_test")
	ctx := testutil.DefaultContext(key, tkey)
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	c := codec.NewProtoCodec(registry)
	k := NewKeeper(
		c,
		runtime.NewKVStoreService(key),
		burnQueryAccountKeeper{moduleAddress: authtypes.NewModuleAddress(types.ModuleName)},
		nil,
		"authority",
	)

	return Querier{Keeper: k}, ctx
}

func TestOngoingProposalsPagination(t *testing.T) {
	querier, ctx := setupBurnQueryKeeper(t)
	for _, proposalID := range []uint64{3, 1, 2} {
		require.NoError(t, querier.OngoingBurnProposals.Set(ctx, proposalID, types.BurnProposal{ProposalId: proposalID}))
	}

	first, err := querier.OngoingProposals(sdk.WrapSDKContext(ctx), &types.QueryOngoingProposalsRequest{Pagination: &query.PageRequest{Limit: 2, CountTotal: true}})
	require.NoError(t, err)
	require.Len(t, first.Proposals, 2)
	require.NotNil(t, first.Pagination)
	require.Equal(t, []uint64{1, 2}, []uint64{first.Proposals[0].ProposalId, first.Proposals[1].ProposalId})
	require.Equal(t, uint64(3), first.Pagination.Total)
	require.NotEmpty(t, first.Pagination.NextKey)

	second, err := querier.OngoingProposals(sdk.WrapSDKContext(ctx), &types.QueryOngoingProposalsRequest{
		Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2},
	})
	require.NoError(t, err)
	require.NotNil(t, second.Pagination)
	require.Len(t, second.Proposals, 1)
	require.Equal(t, uint64(3), second.Proposals[0].ProposalId)
	require.Empty(t, second.Pagination.NextKey)
}
