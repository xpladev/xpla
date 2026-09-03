package keeper

import (
	"bytes"
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/x/volunteer/types"
)

func setupVolunteerQueryKeeper(t *testing.T) (Querier, sdk.Context) {
	t.Helper()

	key := storetypes.NewKVStoreKey(types.StoreKey)
	tkey := storetypes.NewTransientStoreKey("volunteer_query_test")
	ctx := testutil.DefaultContext(key, tkey)
	registry := cdctypes.NewInterfaceRegistry()
	types.RegisterInterfaces(registry)
	c := codec.NewProtoCodec(registry)
	k := NewKeeper(runtime.NewKVStoreService(key), c, nil, nil, "authority")

	return Querier{Keeper: k}, ctx
}

func TestVolunteerValidatorsPaginationIsDeterministic(t *testing.T) {
	querier, ctx := setupVolunteerQueryKeeper(t)
	addresses := make(map[byte]sdk.ValAddress, 3)
	for _, marker := range []byte{3, 1, 2} {
		address := sdk.ValAddress(bytes.Repeat([]byte{marker}, 20))
		addresses[marker] = address
		require.NoError(t, querier.SetVolunteerValidator(ctx, address, types.NewVolunteerValidator(address, int64(marker))))
	}

	first, err := querier.VolunteerValidators(sdk.WrapSDKContext(ctx), &types.QueryVolunteerValidatorsRequest{Pagination: &query.PageRequest{Limit: 2, CountTotal: true}})
	require.NoError(t, err)
	require.NotNil(t, first.Pagination)
	require.Equal(t, []string{addresses[1].String(), addresses[2].String()}, first.VolunteerValidators)
	require.Equal(t, uint64(3), first.Pagination.Total)
	require.NotEmpty(t, first.Pagination.NextKey)

	second, err := querier.VolunteerValidators(sdk.WrapSDKContext(ctx), &types.QueryVolunteerValidatorsRequest{
		Pagination: &query.PageRequest{Key: first.Pagination.NextKey, Limit: 2},
	})
	require.NoError(t, err)
	require.NotNil(t, second.Pagination)
	require.Equal(t, []string{addresses[3].String()}, second.VolunteerValidators)
	require.Empty(t, second.Pagination.NextKey)
}
