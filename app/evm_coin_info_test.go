package app

import (
	"context"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store/rootmulti"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/stretchr/testify/require"

	xplatypes "github.com/xpladev/xpla/types"
)

func TestEVMBaseFeeHistoricalCoinInfo(t *testing.T) {
	xpla := newTestApp(t)
	t.Cleanup(func() { require.NoError(t, xpla.Close()) })

	xplaCoinInfo := vmtypes.EvmCoinInfo{
		Denom:         xplatypes.DefaultDenom,
		ExtendedDenom: xplatypes.DefaultDenom,
		DisplayDenom:  "xpla",
		Decimals:      uint32(vmtypes.EighteenDecimals),
	}
	storedCoinInfo := vmtypes.EvmCoinInfo{
		Denom:         "utest",
		ExtendedDenom: "atest",
		DisplayDenom:  "test",
		Decimals:      uint32(vmtypes.SixDecimals),
	}

	// Commit real versions of the app's stores. All versions have EVM and
	// feemarket state; only the first predates storage of the CoinInfo key.
	versions := []struct {
		coinInfo *vmtypes.EvmCoinInfo
		baseFee  string
	}{
		{nil, "280000000000"},
		{&storedCoinInfo, "0.25"},
		{&xplaCoinInfo, "310000000000"},
	}
	for i, version := range versions {
		header := tmproto.Header{Height: int64(i + 1), Time: time.Unix(int64(i+1), 0).UTC()}
		ctx := xpla.NewUncachedContext(false, header)
		evmParams := vmtypes.DefaultParams()
		evmParams.EvmDenom = xplatypes.DefaultDenom
		require.NoError(t, xpla.EvmKeeper.SetParams(ctx, evmParams))
		feeParams := feemarkettypes.DefaultParams()
		feeParams.BaseFee = sdkmath.LegacyMustNewDecFromStr(version.baseFee)
		require.NoError(t, xpla.FeeMarketKeeper.SetParams(ctx, feeParams))
		if version.coinInfo != nil {
			require.NoError(t, xpla.EvmKeeper.SetEvmCoinInfo(ctx, *version.coinInfo))
		}
		xpla.CommitMultiStore().(*rootmulti.Store).SetCommitHeader(header)
		require.Equal(t, header.Height, xpla.CommitMultiStore().Commit().Version)
	}

	for _, tc := range []struct {
		name        string
		height      int64
		stored      bool
		coinInfo    vmtypes.EvmCoinInfo
		wantBaseFee string
	}{
		{"historical missing CoinInfo", 1, false, xplaCoinInfo, "280000000000"},
		{"historical stored CoinInfo takes precedence", 2, true, storedCoinInfo, "250000000000"},
		{"latest explicit height", 3, true, xplaCoinInfo, "310000000000"},
		{"latest default height", 0, true, xplaCoinInfo, "310000000000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// This is the registered query used by the EVM RPC backend. Supplying
			// a height must read that committed version, not the current store.
			res, err := xpla.Query(context.Background(), &abci.RequestQuery{
				Path:   "/cosmos.evm.vm.v1.Query/BaseFee",
				Data:   mustMarshalProto(t, &vmtypes.QueryBaseFeeRequest{}),
				Height: tc.height,
			})
			require.NoError(t, err)
			require.Equal(t, uint32(0), res.Code, res.Log)
			var baseFee vmtypes.QueryBaseFeeResponse
			require.NoError(t, xpla.AppCodec().Unmarshal(res.Value, &baseFee))
			require.NotNil(t, baseFee.BaseFee)
			require.Equal(t, tc.wantBaseFee, baseFee.BaseFee.String())

			queryCtx, err := xpla.CreateQueryContext(res.Height, false)
			require.NoError(t, err)
			evmStore := queryCtx.KVStore(xpla.GetKey(vmtypes.StoreKey))
			require.Equal(t, tc.stored, evmStore.Has(vmtypes.KeyPrefixEvmCoinInfo))
			before := append([]byte(nil), evmStore.Get(vmtypes.KeyPrefixEvmCoinInfo)...)
			require.Equal(t, tc.coinInfo, xpla.EvmKeeper.GetEvmCoinInfo(queryCtx))
			direct, err := xpla.EvmKeeper.BaseFee(queryCtx, &vmtypes.QueryBaseFeeRequest{})
			require.NoError(t, err)
			require.NotNil(t, direct.BaseFee)
			require.Equal(t, tc.wantBaseFee, direct.BaseFee.String())
			// Inspect the same query cache so an accidental write cannot be
			// hidden by ABCI discarding its cache after the request.
			require.Equal(t, before, evmStore.Get(vmtypes.KeyPrefixEvmCoinInfo))
		})
	}
}
