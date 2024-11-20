package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// HasAccount implements AccountKeeperI.
func (ak AccountKeeper) HasAccount(ctx context.Context, addr sdk.AccAddress) bool {
	addr = ak.getSliceAddress(ctx, addr)
	has, err := ak.Accounts.Has(ctx, addr)
	if err != nil {
		return false
	}

	return has
}

// GetAccount implements AccountKeeperI.
func (ak AccountKeeper) GetAccount(ctx context.Context, addr sdk.AccAddress) (acc sdk.AccountI) {
	addr = ak.getSliceAddress(ctx, addr)
	return ak.AccountKeeper.GetAccount(ctx, addr)
}

// GetSequence Returns the Sequence of the account at address
func (ak AccountKeeper) GetSequence(ctx context.Context, addr sdk.AccAddress) (uint64, error) {
	acc := ak.GetAccount(ctx, addr)
	if acc == nil {
		return 0, errorsmod.Wrapf(sdkerrors.ErrUnknownAddress, "account %s does not exist", addr)
	}

	return acc.GetSequence(), nil
}

// SetAccount implements AccountKeeperI.
func (ak AccountKeeper) SetAccount(ctx context.Context, acc sdk.AccountI) {
	address := acc.GetAddress()
	if len(address) != 20 {
		sliceAddress := address[len(address)-20:]
		ak.SliceAddresses.Set(ctx, sliceAddress, address)
	}
	ak.AccountKeeper.SetAccount(ctx, acc)
}

func (ak AccountKeeper) getSliceAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccAddress {
	originalAddress, err := ak.SliceAddresses.Get(ctx, addr)
	if err != nil {
		return addr
	}

	return originalAddress
}
