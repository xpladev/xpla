package types

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

type ProposalType string

const (
	ProposalTypeRegisterXatp   ProposalType = "RegisterXatp"
	ProposalTypeUnregisterXatp ProposalType = "UnregisterXatp"
)

var (
	_ govtypes.Content = &RegisterXatpProposal{}
	_ govtypes.Content = &UnregisterXatpProposal{}
)

func init() {
	govtypes.RegisterProposalType(string(ProposalTypeRegisterXatp))
	govtypes.RegisterProposalType(string(ProposalTypeUnregisterXatp))
	govtypes.RegisterProposalTypeCodec(&RegisterXatpProposal{}, "xatp/RegisterXatpProposal")
	govtypes.RegisterProposalTypeCodec(&UnregisterXatpProposal{}, "xatp/UnregisterXatpProposal")
}

func NewRegisterXatpProposal(title, description, token, pair, denom string, decimals int) *RegisterXatpProposal {
	return &RegisterXatpProposal{title, description, &XATP{denom, token, pair, int32(decimals)}}
}

// GetTitle returns the title of a register xatp proposal.
func (p *RegisterXatpProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a register xatp proposal.
func (p *RegisterXatpProposal) GetDescription() string { return p.Description }

func (p RegisterXatpProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p RegisterXatpProposal) ProposalType() string { return string(ProposalTypeRegisterXatp) }

// String implements the Stringer interface.
func (p RegisterXatpProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Register XATP Proposal:
  Title:       %s
  Description: %s
  Token:	   %s
  Pair:		   %s
  Denom:	   %s
  Decimals:	   %d
`, p.Title, p.Description, p.Xatp.Token, p.Xatp.Pair, p.Xatp.Denom, p.Xatp.Decimals))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *RegisterXatpProposal) ValidateBasic() error {

	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	if _, err := sdk.AccAddressFromBech32(p.Xatp.Pair); err != nil {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "XATP pair")
	}

	if _, err := sdk.AccAddressFromBech32(p.Xatp.Token); err != nil {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "XATP token")
	}

	if p.Xatp.Denom == "" {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "XATP denom")
	}

	if p.Xatp.Decimals < 0 || p.Xatp.Decimals > 18 {
		return sdkerrors.Wrap(govtypes.ErrInvalidProposalContent, "XATP decimals")
	}

	return nil
}

func NewUnregisterXatpProposal(title, description, denom string) *UnregisterXatpProposal {
	return &UnregisterXatpProposal{title, description, denom}
}

// GetTitle returns the title of a unregister xatp proposal.
func (p *UnregisterXatpProposal) GetTitle() string { return p.Title }

// GetDescription returns the description of a unregister xatp proposal.
func (p *UnregisterXatpProposal) GetDescription() string { return p.Description }

func (p UnregisterXatpProposal) ProposalRoute() string { return RouterKey }

// ProposalType returns the type
func (p UnregisterXatpProposal) ProposalType() string { return string(ProposalTypeUnregisterXatp) }

// String implements the Stringer interface.
func (p UnregisterXatpProposal) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf(`Unregister XATP Proposal:
  Title:       %s
  Description: %s
  Denom:	   %s
`, p.Title, p.Description, p.Denom))
	return b.String()
}

// ValidateBasic validates the proposal
func (p *UnregisterXatpProposal) ValidateBasic() error {

	if err := govtypes.ValidateAbstract(p); err != nil {
		return err
	}

	if p.Denom == "" {
		return sdkerrors.Wrap(sdkerrors.ErrUnpackAny, "XATP denom")
	}

	return nil
}
