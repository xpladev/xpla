package types

import (
	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

const (
	TypeMsgRegisterVolunteerValidator   = "register_volunteer_validator"
	TypeMsgUnregisterVolunteerValidator = "unregister_volunteer_validator"
)

var (
	_ sdk.Msg                            = (*MsgRegisterVolunteerValidator)(nil)
	_ codectypes.UnpackInterfacesMessage = (*MsgRegisterVolunteerValidator)(nil)
	_ sdk.Msg                            = (*MsgUnregisterVolunteerValidator)(nil)
)

func NewMsgRegisterVolunteerValidator(title, description string, delAddr sdk.AccAddress, valAddr sdk.ValAddress, pubKey cryptotypes.PubKey,
	selfDelegation sdk.Coin, validatorDescription stakingtypes.Description) (*RegisterVolunteerValidatorProposal, error) {
	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}
	return &RegisterVolunteerValidatorProposal{
		Title:                title,
		Description:          description,
		ValidatorDescription: validatorDescription,
		Amount:               selfDelegation,
		DelegatorAddress:     delAddr.String(),
		ValidatorAddress:     valAddr.String(),
		Pubkey:               pkAny,
	}, nil
}

func (msg MsgRegisterVolunteerValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	// note that unmarshaling from bech32 ensures either empty or valid
	delAddr, err := sdk.AccAddressFromBech32(msg.DelegatorAddress)
	if err != nil {
		return err
	}
	if delAddr.Empty() {
		return stakingtypes.ErrEmptyDelegatorAddr
	}

	if msg.ValidatorAddress == "" {
		return stakingtypes.ErrEmptyValidatorAddr
	}

	valAddr, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return err
	}
	if !sdk.AccAddress(valAddr).Equals(delAddr) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "validator address is invalid")
	}

	if msg.Pubkey == nil {
		return stakingtypes.ErrEmptyValidatorPubKey
	}

	if !msg.Amount.IsValid() || !msg.Amount.IsPositive() {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "invalid delegation amount")
	}

	if msg.ValidatorDescription == (stakingtypes.Description{}) {
		return errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "empty description")
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (msg MsgRegisterVolunteerValidator) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(msg.Pubkey, &pubKey)
}

func (p MsgRegisterVolunteerValidator) ToCreateValidator() stakingtypes.MsgCreateValidator {
	return stakingtypes.MsgCreateValidator{
		ValidatorAddress:  p.ValidatorAddress,
		DelegatorAddress:  p.DelegatorAddress,
		MinSelfDelegation: sdkmath.OneInt(),
		Pubkey:            p.Pubkey,
		Value:             p.Amount,
		Description:       p.ValidatorDescription,
		Commission:        stakingtypes.NewCommissionRates(sdkmath.LegacyOneDec(), sdkmath.LegacyOneDec(), sdkmath.LegacyZeroDec()),
	}
}

func (msg MsgUnregisterVolunteerValidator) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(msg.Authority); err != nil {
		return sdkerrors.ErrInvalidAddress.Wrapf("invalid authority address: %s", err)
	}

	if msg.ValidatorAddress == "" {
		return stakingtypes.ErrEmptyValidatorAddr
	}

	_, err := sdk.ValAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		return err
	}

	return nil
}
