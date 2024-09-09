package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/xpla/x/reward/types"
)

type Keeper struct {
	storeService store.KVStoreService
	cdc      codec.BinaryCodec
	// the address capable of executing a MsgUpdateParams message. Typically, this
	// should be the x/gov module account.
	authority string

	authKeeper    types.AccountKeeper
	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
	distKeeper    types.DistributionKeeper
	mintKeeper    types.MintKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, storeService store.KVStoreService,
	ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper, dk types.DistributionKeeper, mk types.MintKeeper,
	authority string,
) Keeper {
	// ensure reward module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	return Keeper{
		storeService:  storeService,
		cdc:           cdc,
		authKeeper:    ak,
		bankKeeper:    bk,
		stakingKeeper: sk,
		distKeeper:    dk,
		mintKeeper:    mk,
		authority:     authority,
	}
}

// GetAuthority returns the x/reward module's authority.
func (k Keeper) GetAuthority() string {
	return k.authority
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx context.Context) log.Logger {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.Logger().With("module", "x/"+types.ModuleName)
}

// FundRewardPool allows an account to directly fund the reward pool fund.
// The amount is added to the reward pool account
// An error is returned if the amount cannot be sent to the module account.
func (k Keeper) FundRewardPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return err
	}

	return nil
}
