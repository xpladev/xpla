package types

import (
	"cosmossdk.io/collections"
)

const (
	// ModuleName defines the module name
	ModuleName = "burn"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName
)

var (
	OngoingBurnProposalsPrefix = collections.NewPrefix("on_going_burn_proposals")
)
