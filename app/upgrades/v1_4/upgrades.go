package v1_4

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/ethereum/go-ethereum/common"
	etherminttypes "github.com/evmos/ethermint/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	stakingkeeper "github.com/xpladev/xpla/x/staking/keeper"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	authKeeper authkeeper.AccountKeeper,
	stakingKeeper *stakingkeeper.Keeper,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {

		migrateBaseAccountToEthAccount(ctx, authKeeper)

		err := migrateDustDelegation(ctx, stakingKeeper)
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}

func migrateBaseAccountToEthAccount(ctx sdk.Context, authKeeper authkeeper.AccountKeeper) {
	authKeeper.IterateAccounts(ctx, func(acc authtypes.AccountI) (stop bool) {
		switch acc := acc.(type) {
		case *authtypes.BaseAccount:
			ethAcc := &etherminttypes.EthAccount{
				BaseAccount: acc,
				CodeHash:    common.BytesToHash(evmtypes.EmptyCodeHash).String(),
			}

			authKeeper.SetAccount(ctx, ethAcc)
		}

		return false
	})
}

func migrateDustDelegation(ctx sdk.Context, stakingKeeper *stakingkeeper.Keeper) error {
	validators := stakingKeeper.GetAllValidators(ctx)
	for _, validator := range validators {
		tolerance, err := validator.SharesFromTokens(sdk.OneInt())
		if err != nil {
			return sdkerrors.Wrapf(sdkerrors.ErrLogic, "validator must have valid share")
		}

		delegations := stakingKeeper.GetValidatorDelegations(ctx, validator.GetOperator())
		for _, delegation := range delegations {
			if delegation.Shares.GTE(tolerance) {
				continue
			}

			_, err := stakingKeeper.Unbond(ctx, delegation.GetDelegatorAddr(), validator.GetOperator(), delegation.GetShares())
			if err != nil {
				return sdkerrors.Wrapf(sdkerrors.ErrLogic, "dust delegation must be unbond")
			}
		}
	}

	return nil
}
