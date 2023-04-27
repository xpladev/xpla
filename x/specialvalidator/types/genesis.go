package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(specialValidators []*SpecialValidator) *GenesisState {
	return &GenesisState{
		SpecialValidators: specialValidators,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		SpecialValidators: []*SpecialValidator{},
	}
}

func ValidateGenesis(gs *GenesisState) error {
	for _, addr := range gs.SpecialValidators {
		if _, err := sdk.ValAddressFromBech32(addr.Address); err != nil {
			return err
		}
	}

	return nil
}
