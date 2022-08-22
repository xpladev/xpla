package keeper

import (
	"fmt"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/xpladev/xpla/x/reward/types"
)

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
	distKeeper    types.DistributionKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	bk types.BankKeeper, sk types.StakingKeeper, dk types.DistributionKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:      key,
		cdc:           cdc,
		paramSpace:    paramSpace,
		bankKeeper:    bk,
		stakingKeeper: sk,
		distKeeper:    dk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) FundRewardPool(ctx sdk.Context, amount sdk.Coin, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoins(ctx, sender, types.DelegateProxyAccount, sdk.NewCoins(amount)); err != nil {
		return err
	}

	validators := k.GetValidators(ctx)

	length := len(validators)

	delegateAmount := amount.Amount.Quo(sdk.NewInt(int64(length)))

	for _, validatorStr := range validators {
		validatorAddress, err := sdk.ValAddressFromBech32(validatorStr)
		if err != nil {
			return err
		}

		validator, found := k.stakingKeeper.GetValidator(ctx, validatorAddress)
		if !found {
			return fmt.Errorf("validator record not found for address: %s", validatorStr)
		}

		_, err = k.stakingKeeper.Delegate(ctx, types.DelegateProxyAccount, delegateAmount, stakingtypes.Unbonded, validator, true)
		if err != nil {
			return err
		}
	}

	return nil
}
