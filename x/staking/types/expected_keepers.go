package types

import (
	"context"

	"cosmossdk.io/core/address"

	sdk "github.com/cosmos/cosmos-sdk/types"

	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"
)

type AccountKeeper interface {
	AddressCodec() address.Codec
}

type BankKeeper interface {
	SendCoinsFromModuleToModule(ctx context.Context, senderPool, recipientPool string, amt sdk.Coins) error
}

type VolunteerKeeper interface {
	GetVolunteerValidators(ctx context.Context) (volunteerValidators map[string]volunteertypes.VolunteerValidator, err error)
}
