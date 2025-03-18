package keeper

import (
	"context"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/xpla/x/volunteer/types"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc          codec.BinaryCodec

	stakingKeeper types.StakingKeeper
	distKeeper    types.DistributionKeeper
	authority     string
}

// NewKeeper constructs a message authorization Keeper
func NewKeeper(storeService store.KVStoreService, cdc codec.BinaryCodec, sk types.StakingKeeper, dk types.DistributionKeeper, authority string) Keeper {
	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		stakingKeeper: sk,
		distKeeper:    dk,
		authority:     authority,
	}
}

// GetAuthority returns the x/volunteer module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}
