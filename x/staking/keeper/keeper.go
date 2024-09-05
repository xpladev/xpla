package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "cosmossdk.io/store/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/staking/types"
)

type Keeper struct {
	*stakingkeeper.Keeper

	storeKey        storetypes.StoreKey
	cdc             codec.BinaryCodec
	bankKeeper      types.BankKeeper
	volunteerKeeper types.VolunteerKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key storetypes.StoreKey, ak stakingtypes.AccountKeeper, bk stakingtypes.BankKeeper,
	govModAddress string, vk types.VolunteerKeeper,
) *Keeper {
	return &Keeper{
		Keeper:          stakingkeeper.NewKeeper(cdc, key, ak, bk, govModAddress),
		storeKey:        key,
		cdc:             cdc,
		bankKeeper:      bk,
		volunteerKeeper: vk,
	}
}
