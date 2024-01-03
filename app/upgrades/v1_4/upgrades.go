package v1_4

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	stakingKeeper *stakingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		validators := stakingKeeper.GetAllValidators(ctx)
		for _, validator := range validators {
			tolerance, err := validator.SharesFromTokens(sdk.OneInt())
			if err != nil {
				return nil, sdkerrors.Wrapf(sdkerrors.ErrLogic, "validator must have valid share")
			}

			delegations := stakingKeeper.GetValidatorDelegations(ctx, validator.GetOperator())
			for _, delegation := range delegations {
				if delegation.Shares.GTE(tolerance) {
					continue
				}

				_, err := stakingKeeper.Unbond(ctx, delegation.GetDelegatorAddr(), validator.GetOperator(), delegation.GetShares())
				if err != nil {
					return nil, sdkerrors.Wrapf(sdkerrors.ErrLogic, "dust delegation must be unbond")
				}
			}
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
