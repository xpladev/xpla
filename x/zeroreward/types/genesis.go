package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewGenesisState(zeroRewardValidators []*ZeroRewardValidator) *GenesisState {
	return &GenesisState{
		ZeroRewardValidators: zeroRewardValidators,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		ZeroRewardValidators: []*ZeroRewardValidator{},
	}
}

func ValidateGenesis(gs *GenesisState) error {
	for _, addr := range gs.ZeroRewardValidators {
		if _, err := sdk.ValAddressFromBech32(addr.Address); err != nil {
			return err
		}
	}

	return nil
}
