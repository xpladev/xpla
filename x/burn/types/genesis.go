package types

// Validate performs basic validation of supply genesis data returning an
// error for any failed validation criteria.
func (gs GenesisState) Validate() error {
	for _, proposal := range gs.OngoingBurnProposals {
		if err := proposal.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// NewGenesisState creates a new genesis state.
func NewGenesisState(burnProposals []BurnProposal) *GenesisState {
	return &GenesisState{
		OngoingBurnProposals: burnProposals,
	}
}

// DefaultGenesisState returns a default bank module genesis state.
func DefaultGenesisState() *GenesisState {
	return NewGenesisState([]BurnProposal{})
}
