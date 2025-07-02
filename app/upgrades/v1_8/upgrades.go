package v1_8

import (
	"context"
	"fmt"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/evm/x/feemarket"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/vm"
	vmkeeper "github.com/cosmos/evm/x/vm/keeper"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/common"

	etherminttypes "github.com/xpladev/ethermint/types"

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

		ctx.Logger().Info("Starting migrate EthAccounts to BaseAccounts")
		migrateEthAccountToBaseAccount(ctx, keepers.AccountKeeper, keepers.EvmKeeper)

		ctx.Logger().Info("Starting migrateSlicedAccount")
		migrateSlicedAccount(ctx, keepers.AccountKeeper)

		vmmodule, ok := mm.Modules[vmtypes.ModuleName].(vm.AppModule)
		if !ok {
			return nil, fmt.Errorf("cannot get module %s", vmtypes.ModuleName)
		}
		ctx.Logger().Info(fmt.Sprintf("Starting migrate vm module version from %d to %d", fromVM[vmtypes.ModuleName], vmmodule.ConsensusVersion()))
		fromVM[vmtypes.ModuleName] = vmmodule.ConsensusVersion()

		fmmodule, ok := mm.Modules[feemarkettypes.ModuleName].(feemarket.AppModule)
		if !ok {
			return nil, fmt.Errorf("cannot get module %s", feemarkettypes.ModuleName)
		}
		ctx.Logger().Info(fmt.Sprintf("Starting migrate feemarket module version from %d to %d", fromVM[feemarkettypes.ModuleName], fmmodule.ConsensusVersion()))
		fromVM[feemarkettypes.ModuleName] = fmmodule.ConsensusVersion()

		ctx.Logger().Info("Starting module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		ctx.Logger().Info("Upgrading v1_8 has completed")
		return vm, nil
	}
}

// Set account's codehash to evm keeper and migrate legacy ethermint account
// to cosmos sdk base account
func migrateEthAccountToBaseAccount(ctx sdk.Context, ak authkeeper.AccountKeeper, ek *vmkeeper.Keeper) {
	ak.IterateAccounts(ctx, func(acc sdk.AccountI) (stop bool) {
		ethAccount, ok := acc.(*etherminttypes.EthAccount)
		if !ok {
			return false
		}

		ak.SetAccount(ctx, ethAccount.BaseAccount)

		codehash := common.HexToHash(ethAccount.CodeHash).Bytes()
		if !vmtypes.IsEmptyCodeHash(codehash) {
			ek.SetCodeHash(ctx, ethAccount.EthAddress().Bytes(), codehash)
		}

		return false
	})
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
