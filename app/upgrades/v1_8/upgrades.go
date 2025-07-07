package v1_8

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"

	"github.com/cosmos/evm/x/feemarket"
	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"
	"github.com/cosmos/evm/x/vm"
	vmkeeper "github.com/cosmos/evm/x/vm/keeper"
	vmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/ethereum/go-ethereum/common"

	"github.com/xpladev/xpla/app/keepers"
	v1_7evmtypes "github.com/xpladev/xpla/app/upgrades/v1_8/legacy/evmtypes"
	v1_7feemarkettypes "github.com/xpladev/xpla/app/upgrades/v1_8/legacy/feemarkettypes"
	etherminttypes "github.com/xpladev/xpla/legacy/ethermint/types"
	authkeeper "github.com/xpladev/xpla/x/auth/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
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
		migrateEvmParams(ctx, keepers.GetKey(vmtypes.StoreKey), cdc)

		fmmodule, ok := mm.Modules[feemarkettypes.ModuleName].(feemarket.AppModule)
		if !ok {
			return nil, fmt.Errorf("cannot get module %s", feemarkettypes.ModuleName)
		}
		ctx.Logger().Info(fmt.Sprintf("Starting migrate feemarket module version from %d to %d", fromVM[feemarkettypes.ModuleName], fmmodule.ConsensusVersion()))
		fromVM[feemarkettypes.ModuleName] = fmmodule.ConsensusVersion()
		migrateFeemarketParams(ctx, keepers.GetKey(feemarkettypes.StoreKey), cdc)

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

// migrate params and add new params
func migrateEvmParams(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		paramsV7 v1_7evmtypes.Params
		params   vmtypes.Params
	)

	store := ctx.KVStore(storeKey)

	v7bz := store.Get(vmtypes.KeyPrefixParams)
	cdc.MustUnmarshal(v7bz, &paramsV7)

	// migrate params
	params.EvmDenom = paramsV7.EvmDenom
	params.ExtraEIPs = paramsV7.ExtraEIPs
	params.AllowUnprotectedTxs = paramsV7.AllowUnprotectedTxs
	// add new params
	params.EVMChannels = vmtypes.DefaultEVMChannels
	params.AccessControl = vmtypes.DefaultAccessControl
	params.ActiveStaticPrecompiles = []string{}

	if err := params.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&params)

	store.Set(vmtypes.KeyPrefixParams, bz)
	return nil
}

// migrate params and converts the base fee from Int to LegacyDec
func migrateFeemarketParams(
	ctx sdk.Context,
	storeKey storetypes.StoreKey,
	cdc codec.BinaryCodec,
) error {
	var (
		store    = ctx.KVStore(storeKey)
		paramsV7 v1_7feemarkettypes.Params
		params   feemarkettypes.Params
	)

	v7bz := store.Get(feemarkettypes.ParamsKey)
	cdc.MustUnmarshal(v7bz, &paramsV7)

	params.NoBaseFee = paramsV7.NoBaseFee
	params.BaseFeeChangeDenominator = paramsV7.BaseFeeChangeDenominator
	params.ElasticityMultiplier = paramsV7.ElasticityMultiplier
	params.EnableHeight = paramsV7.EnableHeight
	params.BaseFee = sdkmath.LegacyNewDecFromInt(paramsV7.BaseFee)
	params.MinGasPrice = paramsV7.MinGasPrice
	params.MinGasMultiplier = paramsV7.MinGasMultiplier

	if err := params.Validate(); err != nil {
		return err
	}

	bz, err := cdc.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(feemarkettypes.ParamsKey, bz)

	return nil
}
