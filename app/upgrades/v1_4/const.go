package v1_4

import (
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
)

const (
	UpgradeName = "v1_4"
)

var (
	AddModules = []string{
		consensustypes.ModuleName,
		crisistypes.ModuleName,
	}
)
