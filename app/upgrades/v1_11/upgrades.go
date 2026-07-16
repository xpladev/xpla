package v1_11

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	errorsmod "cosmossdk.io/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/app/keepers"
	xplaprecompile "github.com/xpladev/xpla/precompile"
	xplatypes "github.com/xpladev/xpla/types"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	_ codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, _ upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		// This runs the ibc-go v11.2 in-place migrations on the legacy ibc-apps
		// stores: PFM 3->4 and rate-limiting 1->2. The PFM migration rejects
		// legacy nonrefundable in-flight packets instead of changing their refund
		// semantics during the upgrade.
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		if err := ApplyEVMV07State(ctx, keepers); err != nil {
			return vm, err
		}

		return vm, nil
	}
}

// ApplyEVMV07State reconciles the live v1.10 EVM state with the v0.7 path.
// Existing on-chain parameter values are preserved unless they must change for
// the upgrade; default preinstalls are only added after conflict checks.
func ApplyEVMV07State(ctx sdk.Context, appKeepers *keepers.AppKeepers) error {
	missingPreinstalls, err := collectMissingDefaultPreinstalls(ctx, appKeepers)
	if err != nil {
		return err
	}

	if err := applyEVMParams(ctx, appKeepers); err != nil {
		return err
	}

	if len(missingPreinstalls) == 0 {
		return nil
	}

	return appKeepers.EvmKeeper.AddPreinstalls(ctx, missingPreinstalls)
}

func applyEVMParams(ctx sdk.Context, appKeepers *keepers.AppKeepers) error {
	params := appKeepers.EvmKeeper.GetParams(ctx)
	if params.EvmDenom != xplatypes.DefaultDenom {
		return fmt.Errorf("unexpected evm denom %q; expected %q", params.EvmDenom, xplatypes.DefaultDenom)
	}

	changed := false
	if params.ExtendedDenomOptions == nil || params.ExtendedDenomOptions.ExtendedDenom == "" {
		params.ExtendedDenomOptions = &evmtypes.ExtendedDenomOptions{
			ExtendedDenom: xplatypes.DefaultDenom,
		}
		changed = true
	} else if params.ExtendedDenomOptions.ExtendedDenom != xplatypes.DefaultDenom {
		return fmt.Errorf("unexpected evm extended denom %q; expected %q", params.ExtendedDenomOptions.ExtendedDenom, xplatypes.DefaultDenom)
	}

	activePrecompiles, precompilesChanged, err := mergeActiveStaticPrecompiles(
		params.ActiveStaticPrecompiles,
		v111ActiveStaticPrecompiles(),
	)
	if err != nil {
		return err
	}
	params.ActiveStaticPrecompiles = activePrecompiles
	changed = changed || precompilesChanged

	if err := xplaprecompile.ValidateActiveStaticPrecompiles(params.ActiveStaticPrecompiles); err != nil {
		return err
	}
	if !changed {
		return nil
	}

	return appKeepers.EvmKeeper.SetParams(ctx, params)
}

func v111ActiveStaticPrecompiles() []string {
	return []string{
		evmtypes.ICS02PrecompileAddress,
	}
}

func mergeActiveStaticPrecompiles(existing []string, required []string) ([]string, bool, error) {
	seen := make(map[common.Address]struct{}, len(existing)+len(required))
	merged := make([]string, 0, len(existing)+len(required))
	changed := false

	for _, address := range existing {
		if !common.IsHexAddress(address) {
			return nil, false, fmt.Errorf("invalid precompile %s", address)
		}

		canonical := common.HexToAddress(address)
		if _, ok := seen[canonical]; ok {
			return nil, false, fmt.Errorf("duplicate precompile %s", address)
		}
		seen[canonical] = struct{}{}
		merged = append(merged, canonical.Hex())
		changed = changed || canonical.Hex() != address
	}

	for _, address := range required {
		if !common.IsHexAddress(address) {
			return nil, false, fmt.Errorf("invalid required precompile %s", address)
		}

		canonical := common.HexToAddress(address)
		if _, ok := seen[canonical]; ok {
			continue
		}
		seen[canonical] = struct{}{}
		merged = append(merged, canonical.Hex())
		changed = true
	}

	if !slices.IsSorted(merged) {
		slices.Sort(merged)
		changed = true
	}

	return merged, changed, nil
}

func collectMissingDefaultPreinstalls(ctx sdk.Context, appKeepers *keepers.AppKeepers) ([]evmtypes.Preinstall, error) {
	missingPreinstalls := make([]evmtypes.Preinstall, 0, len(evmtypes.DefaultPreinstalls))
	for _, preinstall := range evmtypes.DefaultPreinstalls {
		code := common.FromHex(preinstall.Code)
		if len(code) == 0 {
			return nil, errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s has no code", preinstall.Address)
		}

		address := common.HexToAddress(preinstall.Address)
		accountAddress := sdk.AccAddress(address.Bytes())
		expectedCodeHash := crypto.Keccak256Hash(code)
		if evmtypes.IsEmptyCodeHash(expectedCodeHash.Bytes()) {
			return nil, errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s has empty code hash", preinstall.Address)
		}

		existingCodeHash := appKeepers.EvmKeeper.GetCodeHash(ctx, address)
		if evmtypes.IsEmptyCodeHash(existingCodeHash.Bytes()) {
			if acc := appKeepers.AccountKeeper.GetAccount(ctx, accountAddress); acc != nil {
				return nil, errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has an account without the expected code hash", preinstall.Address)
			}

			missingPreinstalls = append(missingPreinstalls, preinstall)
			continue
		}

		if !bytes.Equal(existingCodeHash.Bytes(), expectedCodeHash.Bytes()) {
			return nil, errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has a code hash with a different code hash", preinstall.Address)
		}
		if existingCode := appKeepers.EvmKeeper.GetCode(ctx, existingCodeHash); !bytes.Equal(existingCode, code) {
			return nil, errorsmod.Wrapf(evmtypes.ErrInvalidPreinstall, "preinstall %s already has the expected code hash but different code", preinstall.Address)
		}

		if acc := appKeepers.AccountKeeper.GetAccount(ctx, accountAddress); acc == nil {
			missingPreinstalls = append(missingPreinstalls, preinstall)
		}
	}

	return missingPreinstalls, nil
}
