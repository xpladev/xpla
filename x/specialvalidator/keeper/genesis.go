package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	for _, specialValidator := range data.SpecialValidators {
		valAddress, err := sdk.ValAddressFromBech32(specialValidator.Address)
		if err != nil {
			panic(err)
		}

		k.SetSpecialValidator(ctx, valAddress, *specialValidator)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	specialValidators := k.GetSpecialValidators(ctx)

	state := []*types.SpecialValidator{}
	for valAddress, validator := range specialValidators {
		state = append(state, &types.SpecialValidator{
			Address:    valAddress,
			Power:      validator.Power,
			IsDeleting: validator.IsDeleting,
		})
	}

	return types.NewGenesisState(state)
}
