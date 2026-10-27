package v1_15_test

import (
	"testing"
	"time"

	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/stretchr/testify/require"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	xplaapp "github.com/xpladev/xpla/app"
	apphelpers "github.com/xpladev/xpla/app/helpers"
	v1_15 "github.com/xpladev/xpla/app/upgrades/v1_15"
)

func TestApplyUpgradeRunsMigrations(t *testing.T) {
	originalHome := xplaapp.DefaultNodeHome
	xplaapp.DefaultNodeHome = t.TempDir()
	t.Cleanup(func() { xplaapp.DefaultNodeHome = originalHome })

	app := apphelpers.Setup(t, "v1-15-upgrade")
	_, err := app.Commit()
	require.NoError(t, err)

	const upgradeHeight int64 = 100
	ctx := app.NewUncachedContext(false, tmproto.Header{
		Height: upgradeHeight,
		Time:   time.Unix(upgradeHeight, 0).UTC(),
	})

	versionMapBefore, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.True(t, app.UpgradeKeeper.HasHandler(v1_15.UpgradeName))

	require.NoError(t, app.UpgradeKeeper.ApplyUpgrade(ctx, upgradetypes.Plan{
		Name:   v1_15.UpgradeName,
		Height: upgradeHeight,
	}))

	versionMapAfter, err := app.UpgradeKeeper.GetModuleVersionMap(ctx)
	require.NoError(t, err)
	require.Equal(t, versionMapBefore, versionMapAfter)
}
