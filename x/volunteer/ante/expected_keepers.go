package ante

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

type VolunteerKeeper interface {
	GetVolunteerValidator(ctx context.Context, valAddress sdk.ValAddress) (types.VolunteerValidator, error)
}
