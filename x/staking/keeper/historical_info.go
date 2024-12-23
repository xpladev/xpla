package keeper

import (
	"context"
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// TrackHistoricalInfo saves the latest historical-info and deletes the oldest
// heights that are below pruning height
func (k Keeper) TrackHistoricalInfo(ctx context.Context) error {
	entryNum, err := k.HistoricalEntries(ctx)
	if err != nil {
		return err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Prune store to ensure we only have parameter-defined historical entries.
	// In most cases, this will involve removing a single historical entry.
	// In the rare scenario when the historical entries gets reduced to a lower value k'
	// from the original value k. k - k' entries must be deleted from the store.
	// Since the entries to be deleted are always in a continuous range, we can iterate
	// over the historical entries starting from the most recent version to be pruned
	// and then return at the first empty entry.
	for i := sdkCtx.BlockHeight() - int64(entryNum); i >= 0; i-- {
		_, err := k.GetHistoricalInfo(ctx, i)
		if err != nil {
			if errors.Is(err, stakingtypes.ErrNoHistoricalInfo) {
				break
			}
			return err
		}
		if err = k.DeleteHistoricalInfo(ctx, i); err != nil {
			return err
		}
	}

	// if there is no need to persist historicalInfo, return
	if entryNum == 0 {
		return nil
	}

	// Create HistoricalInfo struct
	lastVals, err := k.GetLastValidators(ctx)
	if err != nil {
		return err
	}

	historicalEntry := stakingtypes.NewHistoricalInfo(sdkCtx.BlockHeader(), stakingtypes.Validators{Validators: lastVals, ValidatorCodec: k.ValidatorAddressCodec()}, k.PowerReduction(ctx))

	// Set latest HistoricalInfo at current height
	return k.SetHistoricalInfo(ctx, sdkCtx.BlockHeight(), &historicalEntry)
}
