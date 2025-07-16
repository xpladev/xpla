package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/store"
	"cosmossdk.io/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/xpla/x/burn/types"
)

type Keeper struct {
	cdc          codec.BinaryCodec
	storeService store.KVStoreService
	bankKeeper   types.BankKeeper
	authority    string

	OngoingBurnProposals collections.Map[uint64, types.BurnProposal]
	Schema               collections.Schema
}

func NewKeeper(
	cdc codec.Codec,
	storeService store.KVStoreService,
	ak types.AccountKeeper,
	bk types.BankKeeper,
	authority string,
) Keeper {
	// ensure burn module account is set
	if addr := ak.GetModuleAddress(types.ModuleName); addr == nil {
		panic(fmt.Sprintf("%s module account has not been set", types.ModuleName))
	}

	sb := collections.NewSchemaBuilder(storeService)
	ongoingBurnProposals := collections.NewMap(sb, types.OngoingBurnProposalsPrefix, "ongoing_burn_proposals", collections.Uint64Key, codec.CollValue[types.BurnProposal](cdc))

	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}

	return Keeper{
		cdc:                  cdc,
		storeService:         storeService,
		bankKeeper:           bk,
		authority:            authority,
		OngoingBurnProposals: ongoingBurnProposals,
		Schema:               schema,
	}
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

// GetAuthority returns the authority address
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetAllOngoingBurnProposals retrieves all ongoing burn proposal
func (k Keeper) GetAllOngoingBurnProposals(ctx context.Context) []types.BurnProposal {
	ongoingBurnProposals := make([]types.BurnProposal, 0)
	k.IterateAllOngoingBurnProposals(ctx, func(metadata types.BurnProposal) bool {
		ongoingBurnProposals = append(ongoingBurnProposals, metadata)
		return false
	})

	return ongoingBurnProposals
}

// IterateAllOngoingBurnProposals iterates over all the ongoing burn proposals and
// provides the proposal to a callback. If true is returned from the
// callback, iteration is halted.
func (k Keeper) IterateAllOngoingBurnProposals(ctx context.Context, cb func(types.BurnProposal) bool) {
	err := k.OngoingBurnProposals.Walk(ctx, nil, func(_ uint64, proposal types.BurnProposal) (stop bool, err error) {
		return cb(proposal), nil
	})
	if err != nil {
		panic(err)
	}
}
