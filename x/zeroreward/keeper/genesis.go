package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func (k Keeper) InitGenesis(ctx sdk.Context, data types.GenesisState) {
	for _, zeroRewardValidator := range data.ZeroRewardValidators {
		valAddress, err := sdk.ValAddressFromBech32(zeroRewardValidator.Address)
		if err != nil {
			panic(err)
		}

		k.SetZeroRewardValidator(ctx, valAddress, *zeroRewardValidator)
	}
}

func (k Keeper) ExportGenesis(ctx sdk.Context) *types.GenesisState {
	zeroRewardValidators := k.GetZeroRewardValidators(ctx)

	state := []*types.ZeroRewardValidator{}
	for valAddress, validator := range zeroRewardValidators {
		state = append(state, &types.ZeroRewardValidator{
			Address:    valAddress,
			Power:      validator.Power,
			IsDeleting: validator.IsDeleting,
		})
	}

	return types.NewGenesisState(state)
}
