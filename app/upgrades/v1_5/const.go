package v1_5

import (
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const (
	UpgradeName = "v1_5"
)

var (
	AddModules = []string{
		consensustypes.ModuleName,
		crisistypes.ModuleName,
	}
)
