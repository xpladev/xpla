package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	volunteertypes "github.com/xpladev/xpla/x/volunteer/types"
)

type BankKeeper interface {
	SendCoinsFromModuleToModule(ctx sdk.Context, senderPool, recipientPool string, amt sdk.Coins) error
}

type VolunteerKeeper interface {
	GetVolunteerValidators(ctx sdk.Context) (volunteerValidators map[string]volunteertypes.VolunteerValidator)
}
