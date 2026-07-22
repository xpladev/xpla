package app_test

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/collections"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/feegrant"
	"github.com/stretchr/testify/require"

	apphelpers "github.com/xpladev/xpla/app/helpers"
)

func TestEndBlockerPrunesLegacyFeeGrantExpirationQueue(t *testing.T) {
	xpla := apphelpers.Setup(t, apphelpers.SimAppChainID)
	blockTime := time.Date(2026, time.July, 22, 7, 46, 37, 325_149_646, time.UTC)
	ctx := xpla.BaseApp.NewNextBlockContext(tmproto.Header{
		ChainID: apphelpers.SimAppChainID,
		Height:  21_008_032,
		Time:    blockTime,
	}).WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	grantee := sdk.AccAddress(bytes.Repeat([]byte{1}, 20))
	granter := sdk.AccAddress(bytes.Repeat([]byte{2}, 20))
	expiration := blockTime.Add(-time.Second)
	queueKey := collections.Join3(expiration, grantee, granter)
	rawQueueKey, err := collections.EncodeKeyWithPrefix(
		xpla.FeeGrantKeeper.FeeAllowanceQueue.GetPrefix(),
		xpla.FeeGrantKeeper.FeeAllowanceQueue.KeyCodec(),
		queueKey,
	)
	require.NoError(t, err)

	grant, err := feegrant.NewGrant(granter, grantee, &feegrant.BasicAllowance{Expiration: &expiration})
	require.NoError(t, err)
	grantKey := collections.Join(grantee, granter)
	require.NoError(t, xpla.FeeGrantKeeper.FeeAllowance.Set(ctx, grantKey, grant))
	ctx.KVStore(xpla.GetKey(feegrant.StoreKey)).Set(rawQueueKey, []byte{})

	_, err = xpla.EndBlocker(ctx)
	require.NoError(t, err)

	hasQueueEntry, err := xpla.FeeGrantKeeper.FeeAllowanceQueue.Has(ctx, queueKey)
	require.NoError(t, err)
	require.False(t, hasQueueEntry)
	hasGrant, err := xpla.FeeGrantKeeper.FeeAllowance.Has(ctx, grantKey)
	require.NoError(t, err)
	require.False(t, hasGrant)
}
