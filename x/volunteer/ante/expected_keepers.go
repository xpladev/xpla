package ante

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

type VolunteerKeeper interface {
	GetVolunteerValidator(ctx sdk.Context, valAddress sdk.ValAddress) (types.VolunteerValidator, bool)
}
