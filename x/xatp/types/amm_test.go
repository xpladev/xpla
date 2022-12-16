package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/xatp/types"
)

func TestNormalXatpPair(t *testing.T) {
	normalNativeTokenAssetInfo := &types.AssetNativeToken{
		Denom: "denom",
	}

	normalTokenAssetInfo := &types.AssetCWToken{
		ContractAddr: "cw20",
	}

	normalPair := types.Pair{
		AssetDecimals: []int{1, 2},
		AssetInfos: []types.AssetInfo{
			types.AssetInfo{
				NativeToken: normalNativeTokenAssetInfo,
			},
			types.AssetInfo{
				Token: normalTokenAssetInfo,
			},
		},
		ContractAddr:   "pair",
		LiquidityToken: "liqudity",
	}

	token, tokenDecimal, err := normalPair.Xatp()
	require.NoError(t, err)
	require.Equal(t, normalTokenAssetInfo, token)
	require.Equal(t, 2, tokenDecimal)

	normalPair = types.Pair{
		AssetDecimals: []int{1, 2},
		AssetInfos: []types.AssetInfo{
			types.AssetInfo{
				Token: normalTokenAssetInfo,
			},
			types.AssetInfo{
				NativeToken: normalNativeTokenAssetInfo,
			},
		},
		ContractAddr:   "pair",
		LiquidityToken: "liqudity",
	}

	token, tokenDecimal, err = normalPair.Xatp()
	require.NoError(t, err)
	require.Equal(t, normalTokenAssetInfo, token)
	require.Equal(t, 1, tokenDecimal)
}

func TestInvalidXatpPair(t *testing.T) {
	normalNativeTokenAssetInfo := &types.AssetNativeToken{
		Denom: "denom",
	}

	nativeTokensPair := types.Pair{
		AssetDecimals: []int{1, 2},
		AssetInfos: []types.AssetInfo{
			types.AssetInfo{
				NativeToken: normalNativeTokenAssetInfo,
			},
			types.AssetInfo{
				NativeToken: normalNativeTokenAssetInfo,
			},
		},
		ContractAddr:   "pair",
		LiquidityToken: "liqudity",
	}

	token, tokenDecimal, err := nativeTokensPair.Xatp()
	require.Error(t, err)
	require.Nil(t, token)
	require.Equal(t, -1, tokenDecimal)

	normalTokenAssetInfo := &types.AssetCWToken{
		ContractAddr: "cw20",
	}

	tokensPair := types.Pair{
		AssetDecimals: []int{1, 2},
		AssetInfos: []types.AssetInfo{
			types.AssetInfo{
				Token: normalTokenAssetInfo,
			},
			types.AssetInfo{
				Token: normalTokenAssetInfo,
			},
		},
		ContractAddr:   "pair",
		LiquidityToken: "liqudity",
	}

	token, tokenDecimal, err = tokensPair.Xatp()
	require.Error(t, err)
	require.Nil(t, token)
	require.Equal(t, -1, tokenDecimal)

}
