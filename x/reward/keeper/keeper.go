package keeper

import (
	"fmt"

	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/xpla/x/reward/types"
)

type Keeper struct {
	storeKey storetypes.StoreKey
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
	cdc codec.BinaryCodec, key storetypes.StoreKey,
	ak types.AccountKeeper, bk types.BankKeeper, sk types.StakingKeeper, dk types.DistributionKeeper, mk types.MintKeeper,
	authority string,
) Keeper {
	// ensure reward module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	return Keeper{
		storeKey:      key,
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
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

// FundRewardPool allows an account to directly fund the reward pool fund.
// The amount is added to the reward pool account
// An error is returned if the amount cannot be sent to the module account.
func (k Keeper) FundRewardPool(ctx sdk.Context, amount sdk.Coins, sender sdk.AccAddress) error {
	if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, sender, types.ModuleName, amount); err != nil {
		return err
	}

	return nil
}
