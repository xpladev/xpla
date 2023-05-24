package keeper

import (
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	tmprotocrypto "github.com/tendermint/tendermint/proto/tendermint/crypto"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func (k Keeper) ZeroRewardValidatorCommissionProcess(ctx sdk.Context) error {
	zeroRewardValidators := k.GetZeroRewardValidators(ctx)

	for strValAddr, _ := range zeroRewardValidators {
		valAddr, err := sdk.ValAddressFromBech32(strValAddr)
		if err != nil {
			return err
		}

		commissions, err := k.distKeeper.WithdrawValidatorCommission(ctx, valAddr)
		if err != nil {
			continue
		}

		err = k.distKeeper.FundCommunityPool(ctx, commissions, sdk.AccAddress(valAddr.Bytes()))
		if err != nil {
			return err
		}
	}

	return nil
}

func (k Keeper) ZeroRewardValidatorUpdates(ctx sdk.Context) (updates []abci.ValidatorUpdate, err error) {
	powerReduction := k.stakingKeeper.PowerReduction(ctx)
	zeroRewardValidators := k.GetZeroRewardValidators(ctx)

	for valAddr, zeroRewardValidator := range zeroRewardValidators {

		valAddress, validator, tmProtoPk, err := k.getValidatorInfo(ctx, valAddr)
		if err != nil {
			return nil, err
		}

		if zeroRewardValidator.IsDeleting { // unregister validator
			if err = k.deleteUnregisterZeroRewardValidator(ctx, &updates, valAddress, validator, tmProtoPk); err != nil {
				return nil, err
			}
		} else if validator.IsJailed() { // jailed
			if err = k.updateJailedZeroRewardValidator(ctx, &updates, zeroRewardValidator, valAddress, validator, tmProtoPk); err != nil {
				return nil, err
			}
		} else if !validator.IsBonded() { // if no unregister and jailed status, it must always be bonded
			if err = k.bondZeroRewardValidator(ctx, zeroRewardValidator, valAddress, validator); err != nil {
				return nil, err
			}
		} else if zeroRewardValidator.Power == 0 {
			k.updateZeroRewardValidatorPower(ctx, &updates, zeroRewardValidator, valAddress, validator, tmProtoPk, powerReduction)
		}

	}

	return updates, nil
}

func (k Keeper) deleteUnregisterZeroRewardValidator(ctx sdk.Context, updates *[]abci.ValidatorUpdate, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey) error {

	// when the zero reward validator is not included in the active set
	if validator.IsBonded() && validator.Tokens.Equal(sdk.ZeroInt()) {
		if _, err := k.beginUnbondingValidator(ctx, validator); err != nil {
			return err
		}

		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  0,
		})
	}

	k.DeleteZeroRewardValidator(ctx, valAddress)

	return nil
}

func (k Keeper) updateJailedZeroRewardValidator(ctx sdk.Context, updates *[]abci.ValidatorUpdate, zeroRewardValidator types.ZeroRewardValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey) error {
	// when the zero reward validator is not included in the active set
	if validator.IsBonded() && zeroRewardValidator.Power != 0 {
		if _, err := k.beginUnbondingValidator(ctx, validator); err != nil {
			return err
		}

		zeroRewardValidator.Power = 0
		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  zeroRewardValidator.Power,
		})

		k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
	}

	return nil
}

func (k Keeper) bondZeroRewardValidator(ctx sdk.Context, zeroRewardValidator types.ZeroRewardValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator) error {
	if _, err := k.bondValidator(ctx, validator); err != nil {
		return err
	}

	zeroRewardValidator.Power = 0
	k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)

	return nil
}

func (k Keeper) updateZeroRewardValidatorPower(ctx sdk.Context, updates *[]abci.ValidatorUpdate, zeroRewardValidator types.ZeroRewardValidator, valAddress sdk.ValAddress, validator stakingtypes.Validator, tmProtoPk tmprotocrypto.PublicKey, powerReduction sdk.Int) {
	prevPower := k.stakingKeeper.GetLastValidatorPower(ctx, valAddress)
	newPower := validator.GetConsensusPower(powerReduction)

	if prevPower != newPower {
		*updates = append(*updates, abci.ValidatorUpdate{
			PubKey: tmProtoPk,
			Power:  newPower,
		})
	}

	zeroRewardValidator.Power = newPower
	k.SetZeroRewardValidator(ctx, valAddress, zeroRewardValidator)
}
