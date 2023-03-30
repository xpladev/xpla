package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xpladev/xpla/x/xatp/types"
)

func TestNormalXatp(t *testing.T) {
	xatpKeeper, ctx := createTestInput(t)

	denom := "CTXT"
	token := "token1"
	pair := "pair1"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	})

	xatp, found := xatpKeeper.GetXatp(ctx, denom)
	require.True(t, found)
	require.Equal(t, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	}, xatp)
}

func TestEmptyXatp(t *testing.T) {
	xatpKeeper, ctx := createTestInput(t)

	xatp, found := xatpKeeper.GetXatp(ctx, "empty")
	require.False(t, found)
	require.Equal(t, xatp, types.XATP{
		Denom: "",
		Token: "",
		Pair:  "",
	})
}

func TestUpdateXatp(t *testing.T) {
	xatpKeeper, ctx := createTestInput(t)

	denom := "CTXT"
	token := "token1"
	pair := "pair1"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	})

	xatp, found := xatpKeeper.GetXatp(ctx, denom)
	require.True(t, found)
	require.Equal(t, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	}, xatp)

	updateToken := "token2"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom,
		Token: updateToken,
		Pair:  pair,
	})

	xatp, found = xatpKeeper.GetXatp(ctx, denom)
	require.True(t, found)
	require.Equal(t, types.XATP{
		Denom: denom,
		Token: updateToken,
		Pair:  pair,
	}, xatp)

	// overwrite
	xatps := xatpKeeper.GetAllXatps(ctx)
	require.Equal(t, len(xatps), 1)

}

func TestGetAllXatp(t *testing.T) {
	xatpKeeper, ctx := createTestInput(t)

	denom := "CTXT"
	token := "token1"
	pair := "pair1"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	})

	denom2 := "CTXT2"
	token2 := "token2"
	pair2 := "pair2"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom2,
		Token: token2,
		Pair:  pair2,
	})

	xatps := xatpKeeper.GetAllXatps(ctx)
	require.Equal(t, len(xatps), 2)
	require.Equal(t, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	}, xatps[0])

	require.Equal(t, types.XATP{
		Denom: denom2,
		Token: token2,
		Pair:  pair2,
	}, xatps[1])
}

func TestDeleteXatp(t *testing.T) {
	xatpKeeper, ctx := createTestInput(t)

	denom := "CTXT"
	token := "token1"
	pair := "pair1"

	xatpKeeper.SetXatp(ctx, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	})

	xatp, found := xatpKeeper.GetXatp(ctx, denom)
	require.True(t, found)
	require.Equal(t, types.XATP{
		Denom: denom,
		Token: token,
		Pair:  pair,
	}, xatp)

	xatpKeeper.DeleteXatp(ctx, denom)
	xatp, found = xatpKeeper.GetXatp(ctx, denom)
	require.False(t, found)
	require.Equal(t, types.XATP{
		Denom: "",
		Token: "",
		Pair:  "",
	}, xatp)

}
