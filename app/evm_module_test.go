package app

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	evmtypes "github.com/cosmos/evm/x/vm/types"
)

func TestEVMUpdateParamsRejectsUnsupportedPrecompile(t *testing.T) {
	xpla := newTestApp(t)
	params := evmtypes.DefaultParams()
	params.ActiveStaticPrecompiles = []string{evmtypes.VestingPrecompileAddress}

	_, err := (xplaEVMMsgServer{MsgServer: xpla.EvmKeeper}).UpdateParams(
		context.Background(),
		&evmtypes.MsgUpdateParams{Params: params},
	)
	require.ErrorContains(t, err, "unsupported active static precompile")
}
