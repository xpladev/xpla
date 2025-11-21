package v1_9

import (
	"context"
	"slices"

	upgradetypes "cosmossdk.io/x/upgrade/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	vmtypes "github.com/cosmos/evm/x/vm/types"

	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(c context.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		ctx := sdk.UnwrapSDKContext(c)

		ctx.Logger().Info("Setting denom metadata...")
		keepers.BankKeeper.SetDenomMetaData(ctx, banktypes.Metadata{
			Description: "november draft description",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    "axpla",
					Exponent: 0,
					Aliases:  nil,
				},
				{
					Denom:    "xpla",
					Exponent: 18,
					Aliases:  nil,
				},
			},
			Base:    "axpla",
			Display: "xpla",
			Name:    "XPLA native coin",
			Symbol:  "XPLA",
			URI:     "_uri",
			URIHash: "_uri_hash",
		})

		ctx.Logger().Info("Initiating EVM coin info...")
		// Initialize EvmCoinInfo in the module store
		if err := keepers.EvmKeeper.InitEvmCoinInfo(ctx); err != nil {
			return nil, err
		}

		ctx.Logger().Info("Modifying EVM Active Static Precompiles...")
		deletedEvidencePrecompileAddress := "0x0000000000000000000000000000000000000807"
		params := keepers.EvmKeeper.GetParams(ctx)
		idx := slices.Index(params.ActiveStaticPrecompiles, deletedEvidencePrecompileAddress)
		if idx != -1 {
			params.ActiveStaticPrecompiles = slices.Delete(params.ActiveStaticPrecompiles, idx, idx+1)
		}
		params.ActiveStaticPrecompiles = append(params.ActiveStaticPrecompiles, vmtypes.ICS20PrecompileAddress)
		if err := keepers.EvmKeeper.SetParams(ctx, params); err != nil {
			return nil, err
		}

		ctx.Logger().Info("Starting module migrations...")
		vm, err := mm.RunMigrations(ctx, configurator, fromVM)
		if err != nil {
			return vm, err
		}

		ctx.Logger().Info("Upgrading v1_9 has completed")
		return vm, nil
	}
}
