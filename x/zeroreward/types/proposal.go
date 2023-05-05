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
	ProposalTypeRegisterZeroRewardValidator   ProposalType = "RegisterZeroRewardValidator"
	ProposalTypeUnregisterZeroRewardValidator ProposalType = "UnregisterZeroRewardValidator"
)

var (
	_ govtypes.Content = &RegisterZeroRewardValidatorProposal{}
	_ govtypes.Content = &UnregisterZeroRewardValidatorProposal{}
)

func init() {
	govtypes.RegisterProposalType(string(ProposalTypeRegisterZeroRewardValidator))
	govtypes.RegisterProposalType(string(ProposalTypeUnregisterZeroRewardValidator))
	govtypes.RegisterProposalTypeCodec(&RegisterZeroRewardValidatorProposal{}, "zeroreward/RegisterZeroRewardValidatorProposal")
	govtypes.RegisterProposalTypeCodec(&UnregisterZeroRewardValidatorProposal{}, "zeroreward/UnregisterZeroRewardValidatorProposal")
}

func NewRegisterZeroRewardValidatorProposal(title, description string, delAddr sdk.AccAddress, valAddr sdk.ValAddress, pubKey cryptotypes.PubKey,
	selfDelegation sdk.Coin, validatorDescription stakingtypes.Description) (*RegisterZeroRewardValidatorProposal, error) {
	var pkAny *codectypes.Any
	if pubKey != nil {
		var err error
		if pkAny, err = codectypes.NewAnyWithValue(pubKey); err != nil {
			return nil, err
		}
	}
	return &RegisterZeroRewardValidatorProposal{
		Title:                title,
		Description:          description,
		ValidatorDescription: validatorDescription,
		Amount:               selfDelegation,
		DelegatorAddress:     delAddr.String(),
		ValidatorAddress:     valAddr.String(),
		Pubkey:               pkAny,
	}, nil
}

// GetTitle returns the title of a register zero reward validator proposal.
func (p *RegisterZeroRewardValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a register zero reward validator proposal.
func (p *RegisterZeroRewardValidatorProposal) GetDescription() string { return p.Description }

func (p RegisterZeroRewardValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p RegisterZeroRewardValidatorProposal) ProposalType() string {
	return string(ProposalTypeRegisterZeroRewardValidator)
}

// String implements the Stringer interface.
func (p RegisterZeroRewardValidatorProposal) String() string {
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
func (p *RegisterZeroRewardValidatorProposal) ValidateBasic() error {

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
func (p RegisterZeroRewardValidatorProposal) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	var pubKey cryptotypes.PubKey
	return unpacker.UnpackAny(p.Pubkey, &pubKey)
}

func (p RegisterZeroRewardValidatorProposal) ToCreateValidator() stakingtypes.MsgCreateValidator {
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

func NewUnregisterZeroRewardValidatorProposal(title, description string, validatorAddress sdk.ValAddress) *UnregisterZeroRewardValidatorProposal {
	return &UnregisterZeroRewardValidatorProposal{
		Title:            title,
		Description:      description,
		ValidatorAddress: validatorAddress.String()}
}

// GetTitle returns the title of a unregister zero reward validator proposal.
func (p *UnregisterZeroRewardValidatorProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a unregister zero reward validator proposal.
func (p *UnregisterZeroRewardValidatorProposal) GetDescription() string { return p.Description }

func (p UnregisterZeroRewardValidatorProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UnregisterZeroRewardValidatorProposal) ProposalType() string {
	return string(ProposalTypeUnregisterZeroRewardValidator)
}

// String implements the Stringer interface.
func (p UnregisterZeroRewardValidatorProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Unregister Zero Reward Validator Proposal:
	Title:	%s
	Description:	%s
	Validator:	%s
`, p.Title, p.Description, p.ValidatorAddress))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *UnregisterZeroRewardValidatorProposal) ValidateBasic() error {

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
