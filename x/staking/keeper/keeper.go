package keeper

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/staking/types"
)

type Keeper struct {
	stakingkeeper.Keeper

	storeKey        sdk.StoreKey
	cdc             codec.BinaryCodec
	bankKeeper      types.BankKeeper
	volunteerKeeper types.VolunteerKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, ak stakingtypes.AccountKeeper, bk stakingtypes.BankKeeper,
	ps paramtypes.Subspace, vk types.VolunteerKeeper,
) Keeper {
	return Keeper{
		Keeper:          stakingkeeper.NewKeeper(cdc, key, ak, bk, ps),
		storeKey:        key,
		cdc:             cdc,
		bankKeeper:      bk,
		volunteerKeeper: vk,
	}
}
