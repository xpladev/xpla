package types

import vmtypes "github.com/cosmos/evm/x/vm/types"

const (
	// Default Denom
	DefaultDenom = "axpla"

	// CoinType
	CoinType = 60

	// FullFundraiserPath is the parts of the BIP44 HD path
	FullFundraiserPath = "m/44'/60'/0'/0/0"

	DefaultDenomPrecision = int64(18)
)

// DefaultXPLACoinInfo returns the default EVM coin metadata for XPLA.
func DefaultXPLACoinInfo() vmtypes.EvmCoinInfo {
	return vmtypes.EvmCoinInfo{
		Denom:         DefaultDenom,
		ExtendedDenom: DefaultDenom,
		DisplayDenom:  "xpla",
		Decimals:      uint32(DefaultDenomPrecision),
	}
}
