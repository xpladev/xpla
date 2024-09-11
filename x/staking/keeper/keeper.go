package keeper

import (
	"cosmossdk.io/core/address"
	"cosmossdk.io/core/store"

	"github.com/cosmos/cosmos-sdk/codec"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/xpladev/xpla/x/staking/types"
)

type Keeper struct {
	*stakingkeeper.Keeper

	storeService    store.KVStoreService
	cdc             codec.BinaryCodec
	authKeeper      types.AccountKeeper
	bankKeeper      types.BankKeeper
	volunteerKeeper types.VolunteerKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, storeService store.KVStoreService, ak stakingtypes.AccountKeeper,
	bk stakingtypes.BankKeeper, govModAddress string, vk types.VolunteerKeeper,
	validatorAddressCodec address.Codec, consensusAddressCodec address.Codec,
) *Keeper {
	return &Keeper{
		Keeper:          stakingkeeper.NewKeeper(cdc, storeService, ak, bk, govModAddress, validatorAddressCodec, consensusAddressCodec),
		storeService:    storeService,
		cdc:             cdc,
		authKeeper:      ak,
		bankKeeper:      bk,
		volunteerKeeper: vk,
	}
}
