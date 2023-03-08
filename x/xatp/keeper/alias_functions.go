package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func (k Keeper) GetXatpPayerAccount() sdk.AccAddress {
	return k.authKeeper.GetModuleAddress(types.ModuleName)
}
