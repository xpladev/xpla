package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetModuleAddress(name string) sdk.AccAddress
}
