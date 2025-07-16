package types

import (
	errorsmod "cosmossdk.io/errors"
)

// x/burn module sentinel errors
var (
	ErrBurnProposalNotFound = errorsmod.Register(ModuleName, 1, "burn proposal not found")
	ErrInvalidBurnAmount    = errorsmod.Register(ModuleName, 2, "invalid burn amount")
	ErrBurnProposalExists   = errorsmod.Register(ModuleName, 3, "burn proposal already exists")
)
