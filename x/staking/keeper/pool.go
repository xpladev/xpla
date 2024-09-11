package keeper

import (
	"context"

	"cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// notBondedTokensToBonded transfers coins from the not bonded to the bonded pool within staking
func (k Keeper) notBondedTokensToBonded(ctx context.Context, tokens math.Int) error {
	bondDenom, err := k.BondDenom(ctx)
	if err != nil {
		return err
	}
	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, tokens))

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.NotBondedPoolName, types.BondedPoolName, coins)
}

// bondedTokensToNotBonded transfers coins from the bonded to the not bonded pool within staking
func (k Keeper) bondedTokensToNotBonded(ctx context.Context, tokens math.Int) error {
	bondDenom, err := k.BondDenom(ctx)
	if err != nil {
		return err
	}

	coins := sdk.NewCoins(sdk.NewCoin(bondDenom, tokens))

	return k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.BondedPoolName, types.NotBondedPoolName, coins)
}
