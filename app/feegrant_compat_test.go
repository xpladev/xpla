package app

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/collections"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/address"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"
)

func TestFeeGrantKeeperAcceptsLegacyExpirationQueueValue(t *testing.T) {
	xpla := newTestApp(t)
	blockTime := time.Date(2026, time.July, 22, 7, 46, 37, 325_149_646, time.UTC)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{Height: 21_008_032, Time: blockTime})

	grantee := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	granter := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	expiration := blockTime.Add(-time.Second)
	queueKey := collections.Join3(expiration, grantee, granter)

	_, err := collections.BoolValue.Decode([]byte{})
	require.ErrorContains(t, err, "wanted size to be at least 1")

	rawQueueKey, err := collections.EncodeKeyWithPrefix(
		xpla.FeeGrantKeeper.FeeAllowanceQueue.GetPrefix(),
		xpla.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(),
		queueKey,
	)
	require.NoError(t, err)
	legacyQueueKey := append([]byte{1}, sdk.FormatTimeBytes(expiration)...)
	legacyQueueKey = append(legacyQueueKey, address.MustLengthPrefix(grantee)...)
	legacyQueueKey = append(legacyQueueKey, address.MustLengthPrefix(granter)...)
	require.Equal(t, legacyQueueKey, rawQueueKey)

	store := ctx.KVStore(xpla.GetKey(feegrant.StoreKey))
	grant, err := feegrant.NewGrant(granter, grantee, &feegrant.BasicAllowance{Expiration: &expiration})
	require.NoError(t, err)
	grantKey := collections.Join(grantee, granter)
	require.NoError(t, xpla.FeeGrantKeeper.FeeAllowance.Set(ctx, grantKey, grant))
	store.Set(legacyQueueKey, []byte{})
	require.NotNil(t, store.Get(rawQueueKey))
	require.Empty(t, store.Get(rawQueueKey))
	hasGrant, err := xpla.FeeGrantKeeper.FeeAllowance.Has(ctx, grantKey)
	require.NoError(t, err)
	require.True(t, hasGrant)

	value, err := xpla.FeeGrantKeeper.FeeAllowanceQueue.Get(ctx, queueKey)
	require.NoError(t, err)
	require.True(t, value)

	require.NoError(t, xpla.FeeGrantKeeper.RemoveExpiredAllowances(ctx, 1))
	require.Nil(t, store.Get(rawQueueKey))
	hasGrant, err = xpla.FeeGrantKeeper.FeeAllowance.Has(ctx, grantKey)
	require.NoError(t, err)
	require.False(t, hasGrant)
}

func TestFeeGrantKeeperWritesCanonicalExpirationQueueValue(t *testing.T) {
	xpla := newTestApp(t)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{Height: 1})

	queueKey := collections.Join3(
		time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		sdk.AccAddress(bytes.Repeat([]byte{1}, 20)),
		sdk.AccAddress(bytes.Repeat([]byte{2}, 20)),
	)
	require.NoError(t, xpla.FeeGrantKeeper.FeeAllowanceQueue.Set(ctx, queueKey, true))

	rawQueueKey, err := collections.EncodeKeyWithPrefix(
		xpla.FeeGrantKeeper.FeeAllowanceQueue.GetPrefix(),
		xpla.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(),
		queueKey,
	)
	require.NoError(t, err)
	require.Equal(t, []byte{1}, ctx.KVStore(xpla.GetKey(feegrant.StoreKey)).Get(rawQueueKey))
}

func TestFeeGrantKeeperRejectsUnknownExpirationQueueValue(t *testing.T) {
	xpla := newTestApp(t)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{Height: 1})

	queueKey := collections.Join3(
		time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC),
		sdk.AccAddress(bytes.Repeat([]byte{1}, 20)),
		sdk.AccAddress(bytes.Repeat([]byte{2}, 20)),
	)
	rawQueueKey, err := collections.EncodeKeyWithPrefix(
		xpla.FeeGrantKeeper.FeeAllowanceQueue.GetPrefix(),
		xpla.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(),
		queueKey,
	)
	require.NoError(t, err)
	ctx.KVStore(xpla.GetKey(feegrant.StoreKey)).Set(rawQueueKey, []byte{2})

	_, err = xpla.FeeGrantKeeper.FeeAllowanceQueue.Get(ctx, queueKey)
	require.ErrorContains(t, err, "invalid legacy feegrant expiration queue value: 02")
}
