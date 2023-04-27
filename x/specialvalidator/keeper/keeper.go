package keeper

import (
	"github.com/tendermint/tendermint/libs/log"
	"github.com/xpladev/xpla/x/specialvalidator/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	storeKey sdk.StoreKey
	cdc      codec.BinaryCodec

	stakingKeeper types.StakingKeeper
	distKeeper    types.DistributionKeeper
}

// NewKeeper constructs a message authorization Keeper
func NewKeeper(storeKey sdk.StoreKey, cdc codec.BinaryCodec, sk types.StakingKeeper, dk types.DistributionKeeper) Keeper {
	return Keeper{
		storeKey:      storeKey,
		cdc:           cdc,
		stakingKeeper: sk,
		distKeeper:    dk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
