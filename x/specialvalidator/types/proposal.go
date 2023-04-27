package types

import (
	"fmt"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type ProposalType string

const (
	ProposalTypeRegisterSpecialValidator   ProposalType = "RegisterSpecialValidator"
	ProposalTypeUnregisterSpecialValidator ProposalType = "UnregisterSpecialValidator"
)

var (
	_ govtypes.Content = &RegisterSpecialValidatorProposal{}
	_ govtypes.Content = &UnregisterSpecialValidatorProposal{}
)

func init() {
	govtypes.RegisterProposalType(string(ProposalTypeRegisterSpecialValidator))
	govtypes.RegisterProposalType(string(ProposalTypeUnregisterSpecialValidator))
	govtypes.RegisterProposalTypeCodec(&RegisterSpecialValidatorProposal{}, "specialvalidator/RegisterSpecialValidatorProposal")
	govtypes.RegisterProposalTypeCodec(&UnregisterSpecialValidatorProposal{}, "specialvalidator/UnregisterSpecialValidatorProposal")
}

func NewRegisterSpecialValidatorProposal(title, description string, delAddr sdk.AccAddress, valAddr sdk.ValAddress, pubKey cryptotypes.PubKey,
	selfDelegation sdk.Coin, validatorDescription stakingtypes.Description) (*RegisterSpecialValidatorProposal, error) {
	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}
	return &RegisterSpecialValidatorProposal{
		Title:                title,
		Description:          description,
		ValidatorDescription: validatorDescription,
		Amount:               selfDelegation,
		DelegatorAddress:     delAddr.String(),
		ValidatorAddress:     valAddr.String(),
		Pubkey:               pkAny,
	}, nil
}

// GetTitle returns the title of a register special validator proposal.
func (p *RegisterSpecialValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a register special validator proposal.
func (p *RegisterSpecialValidatorProposal) GetDescription() string { return p.Description }

func (p RegisterSpecialValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p RegisterSpecialValidatorProposal) ProposalType() string {
	return string(ProposalTypeRegisterSpecialValidator)
}

// String implements the Stringer interface.
func (p RegisterSpecialValidatorProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Register Special Validator Proposal:
	Title:	%s
	Description:	%s
	Delegator:	%s
	Validator:	%s
	Pubkey:	%s
	ValidatorDescription:	%s
	Amount:	%s
`, p.Title, p.Description, p.DelegatorAddress, p.ValidatorAddress, p.Pubkey.String(), p.ValidatorDescription.String(), p.Amount.String()))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *RegisterSpecialValidatorProposal) ValidateBasic() error {

	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	// note that unmarshaling from bech32 ensures either empty or valid
	delAddr, err := sdk.AccAddressFromBech32(p.DelegatorAddress)
	if err != nil {
		return err
	}
	if delAddr.Empty() {
		return stakingtypes.ErrEmptyDelegatorAddr
	}

	if p.ValidatorAddress == "" {
		return stakingtypes.ErrEmptyValidatorAddr
	}

	valAddr, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}
	if !sdk.AccAddress(valAddr).Equals(delAddr) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "validator address is invalid")
	}

	if p.Pubkey == nil {
		return stakingtypes.ErrEmptyValidatorPubKey
	}

	if !p.Amount.IsValid() || !p.Amount.IsPositive() {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "invalid delegation amount")
	}

	if p.ValidatorDescription == (stakingtypes.Description{}) {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "empty description")
	}

	return nil
}

// UnpackInterfaces implements UnpackInterfacesMessage.UnpackInterfaces
func (p RegisterSpecialValidatorProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(p.Pubkey, &pubKey)
}

func (p RegisterSpecialValidatorProposal) ToCreateValidator() stakingtypes.MsgCreateValidator {
	return stakingtypes.MsgCreateValidator{
		ValidatorAddress:  p.ValidatorAddress,
		DelegatorAddress:  p.DelegatorAddress,
		MinSelfDelegation: sdk.OneInt(),
		Pubkey:            p.Pubkey,
		Value:             p.Amount,
		Description:       p.ValidatorDescription,
		Commission:        stakingtypes.NewCommissionRates(sdk.OneDec(), sdk.OneDec(), sdk.ZeroDec()),
	}
}

func NewUnregisterSpecialValidatorProposal(title, description string, validatorAddress sdk.ValAddress) *UnregisterSpecialValidatorProposal {
	return &UnregisterSpecialValidatorProposal{
		Title:            title,
		Description:      description,
		ValidatorAddress: validatorAddress.String()}
}

// GetTitle returns the title of a unregister special validator proposal.
func (p *UnregisterSpecialValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a unregister special validator proposal.
func (p *UnregisterSpecialValidatorProposal) GetDescription() string { return p.Description }

func (p UnregisterSpecialValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UnregisterSpecialValidatorProposal) ProposalType() string {
	return string(ProposalTypeUnregisterSpecialValidator)
}

// String implements the Stringer interface.
func (p UnregisterSpecialValidatorProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Unregister Special Validator Proposal:
	Title:	%s
	Description:	%s
	Validator:	%s
`, p.Title, p.Description, p.ValidatorAddress))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *UnregisterSpecialValidatorProposal) ValidateBasic() error {

	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	if p.ValidatorAddress == "" {
		return stakingtypes.ErrEmptyValidatorAddr
	}

	_, err := sdk.ValAddressFromBech32(p.ValidatorAddress)
	if err != nil {
		return err
	}

	return nil
}
