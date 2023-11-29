package volunteercube

import (
	erc20types "github.com/evmos/evmos/v9/x/erc20/types"
)

const (
	UpgradeName = "VolunteerCube"
)

var (
	AddModules = []string{
		erc20types.ModuleName,
	}
)
