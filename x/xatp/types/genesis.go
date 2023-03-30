package types

func NewGenesisState(params Params, xatps []XATP) *GenesisState {
	return &GenesisState{
		Params: params,
		Xatps:  xatps,
	}
}

func DefaultGenesisState() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Xatps:  []XATP{},
	}
}

func ValidateGenesis(gs *GenesisState) error {
	return gs.Params.ValidateBasic()
}
