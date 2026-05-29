package v1_10

import (
	"bytes"
	"context"

	errorsmod "cosmossdk.io/errors"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/app/keepers"
	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		if err := ApplyEVMDefaults(ctx, keepers); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// ApplyEVMDefaults updates EVM params and installs missing default preinstalls for the v1_10 upgrade.
func ApplyEVMDefaults(ctx sdk.Context, appKeepers *keepers.AppKeepers) error {
	evmParams := appKeepers.EvmKeeper.GetParams(ctx)
	evmParams.EvmDenom = xplatypes.DefaultDenom
	evmParams.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
		ExtendedDenom: xplatypes.DefaultDenom,
	}
	evmParams.ActiveStaticPrecompiles = xplaprecompile.DefaultActiveStaticPrecompiles()
	if err := appKeepers.EvmKeeper.SetParams(ctx, evmParams); err != nil {
		return err
	}

	return addMissingDefaultPreinstalls(ctx, appKeepers)
}

func addMissingDefaultPreinstalls(ctx sdk.Context, appKeepers *keepers.AppKeepers) error {
	missingPreinstalls := make([]evmtypes.Preinstall, 0, len(evmtypes.DefaultPreinstalls))
	for _, preinstall := range evmtypes.DefaultPreinstalls {
		address := common.HexToAddress(preinstall.Address)
		accountAddress := sdk.AccAddress(address.Bytes())
		code := common.FromHex(preinstall.Code)
		if len(code) == 0 {
			return errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s has no code", preinstall.Address)
		}

		expectedCodeHash := crypto.Keccak256Hash(code)
		if evmtypes.IsEmptyCodeHash(expectedCodeHash.Bytes()) {
			return errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s has empty code hash", preinstall.Address)
		}

		existingCodeHash := appKeepers.EvmKeeper.GetCodeHash(ctx, address)
		if evmtypes.IsEmptyCodeHash(existingCodeHash.Bytes()) {
			if acc := appKeepers.AccountKeeper.GetAccount(ctx, accountAddress); acc != nil {
				return errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has an account without the expected code hash", preinstall.Address)
			}

			missingPreinstalls = append(missingPreinstalls, preinstall)
			continue
		}

		if !bytes.Equal(existingCodeHash.Bytes(), expectedCodeHash.Bytes()) {
			return errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has a code hash with a different code hash", preinstall.Address)
		}
		if existingCode := appKeepers.EvmKeeper.GetCode(ctx, existingCodeHash); !bytes.Equal(existingCode, code) {
			return errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has the expected code hash but different code", preinstall.Address)
		}

		if acc := appKeepers.AccountKeeper.GetAccount(ctx, accountAddress); acc == nil {
			missingPreinstalls = append(missingPreinstalls, preinstall)
		}
	}

	if len(missingPreinstalls) == 0 {
		return nil
	}

	return appKeepers.EvmKeeper.AddPreinstalls(ctx, missingPreinstalls)
}
