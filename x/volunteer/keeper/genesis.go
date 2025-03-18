package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	for _, volunteerValidator := range data.VolunteerValidators {
		valAddress, err := sdk.ValAddressFromBech32(volunteerValidator.Address)
		if err != nil {
			panic(err)
		}

		k.SetVolunteerValidator(ctx, valAddress, *volunteerValidator)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	volunteerValidators, err := k.GetVolunteerValidators(ctx)
	if err != nil {
		panic(err)
	}

	state := []*types.VolunteerValidator{}
	for valAddress, validator := range volunteerValidators {
		state = append(state, &types.VolunteerValidator{
			Address: valAddress,
			Power:   validator.Power,
		})
	}

	return types.NewGenesisState(state)
}
