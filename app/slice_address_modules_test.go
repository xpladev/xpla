package app

import (
	"bytes"
	"testing"
	"time"

	"cosmossdk.io/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"
)

func TestEveryConfiguredModuleAddressIsExcludedFromWasmAliases(t *testing.T) {
	app := NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		t.TempDir(),
		EmptyAppOptions{},
		EmptyWasmOptions,
		baseapp.SetChainID("module-alias-test"),
	)
	t.Cleanup(func() { require.NoError(t, app.Close()) })
	ctx := app.NewUncachedContext(false, tmproto.Header{
		ChainID: "module-alias-test",
		Height:  1,
		Time:    time.Unix(1, 0).UTC(),
	})

	for moduleName := range maccPerms {
		t.Run(moduleName, func(t *testing.T) {
			moduleAddress := authtypes.NewModuleAddress(moduleName)
			contractAddress := sdk.AccAddress(append(bytes.Repeat([]byte{0x90}, 12), moduleAddress...))
			app.AccountKeeper.SetAccount(ctx, app.AccountKeeper.NewAccountWithAddress(ctx, contractAddress))
			require.ErrorContains(
				t,
				app.WasmKeeper.RegisterWasmAlias(ctx, moduleAddress, contractAddress),
				"protected module",
			)
		})
	}
}
