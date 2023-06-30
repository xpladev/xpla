package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(volunteerValidators []*VolunteerValidator) *GenesisState {
	return &GenesisState{
		VolunteerValidators: volunteerValidators,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		VolunteerValidators: []*VolunteerValidator{},
	}
}

func ValidateGenesis(gs *GenesisState) error {
	for _, addr := range gs.VolunteerValidators {
		if _, err := sdk.ValAddressFromBech32(addr.Address); err != nil {
			return err
		}
	}

	return nil
}
