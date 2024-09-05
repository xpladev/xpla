package keeper

import (
	"cosmossdk.io/log"
	"github.com/xpladev/xpla/x/volunteer/types"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
	cdc      codec.BinaryCodec

	stakingKeeper types.StakingKeeper
	distKeeper    types.DistributionKeeper
	authority     string
}

// NewKeeper constructs a message authorization Keeper
func NewKeeper(storeKey storetypes.StoreKey, cdc codec.BinaryCodec, sk types.StakingKeeper, dk types.DistributionKeeper, authority string) Keeper {
	return Keeper{
		storeKey:      storeKey,
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
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}
