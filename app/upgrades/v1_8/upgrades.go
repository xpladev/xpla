package v1_8

import (
	"context"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/xpladev/xpla/app/keepers"
	authkeeper "github.com/xpladev/xpla/x/auth/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		ctx.Logger().Info("Starting migrateSlicedAccount")
		migrateSlicedAccount(ctx, keepers.AccountKeeper)

		ctx.Logger().Info("Starting module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		ctx.Logger().Info("Upgrading v1_8 has completed")
		return vm, nil
	}
}

// If the length of the address exceeds 20 (for example, a cosmwasm address),
// set the sliceAddress for it.
func migrateSlicedAccount(ctx sdk.Context, ak authkeeper.AccountKeeper) {
	ak.IterateAccounts(ctx, func(acc sdk.AccountI) (stop bool) {
		address := acc.GetAddress()
		if len(address) != 20 {
			sliceAddress := address[len(address)-20:]
			ak.SliceAddresses.Set(ctx, sliceAddress, address)
		}

		return false
	})
}
