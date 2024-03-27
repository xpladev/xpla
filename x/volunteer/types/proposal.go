package types

import (
	"fmt"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type ProposalType string

const (
	ProposalTypeRegisterVolunteerValidator   ProposalType = "RegisterVolunteerValidator"
	ProposalTypeUnregisterVolunteerValidator ProposalType = "UnregisterVolunteerValidator"
)

var (
	_ govtypes.Content = &RegisterVolunteerValidatorProposal{}
	_ govtypes.Content = &UnregisterVolunteerValidatorProposal{}
)

func init() {
	govtypes.RegisterProposalType(string(ProposalTypeRegisterVolunteerValidator))
	govtypes.RegisterProposalType(string(ProposalTypeUnregisterVolunteerValidator))
}

func NewRegisterVolunteerValidatorProposal(title, description string, delAddr sdk.AccAddress, valAddr sdk.ValAddress, pubKey cryptotypes.PubKey,
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

// GetTitle returns the title of a register volunteer validator proposal.
func (p *RegisterVolunteerValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a register volunteer validator proposal.
func (p *RegisterVolunteerValidatorProposal) GetDescription() string { return p.Description }

func (p RegisterVolunteerValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p RegisterVolunteerValidatorProposal) ProposalType() string {
	return string(ProposalTypeRegisterVolunteerValidator)
}

// String implements the Stringer interface.
func (p RegisterVolunteerValidatorProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Register Zero Reward Validator Proposal:
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
func (p *RegisterVolunteerValidatorProposal) ValidateBasic() error {

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
func (p RegisterVolunteerValidatorProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(p.Pubkey, &pubKey)
}

func (p RegisterVolunteerValidatorProposal) ToCreateValidator() stakingtypes.MsgCreateValidator {
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

func NewUnregisterVolunteerValidatorProposal(title, description string, validatorAddress sdk.ValAddress) *UnregisterVolunteerValidatorProposal {
	return &UnregisterVolunteerValidatorProposal{
		Title:            title,
		Description:      description,
		ValidatorAddress: validatorAddress.String()}
}

// GetTitle returns the title of a unregister volunteer validator proposal.
func (p *UnregisterVolunteerValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a unregister volunteer validator proposal.
func (p *UnregisterVolunteerValidatorProposal) GetDescription() string { return p.Description }

func (p UnregisterVolunteerValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UnregisterVolunteerValidatorProposal) ProposalType() string {
	return string(ProposalTypeUnregisterVolunteerValidator)
}

// String implements the Stringer interface.
func (p UnregisterVolunteerValidatorProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Unregister Zero Reward Validator Proposal:
	Title:	%s
	Description:	%s
	Validator:	%s
`, p.Title, p.Description, p.ValidatorAddress))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *UnregisterVolunteerValidatorProposal) ValidateBasic() error {

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
