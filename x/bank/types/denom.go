package types

import (
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
)

const (
	ERC20          = "xerc20"
	CW20           = "xcw20"
	TYPE_SEPARATOR = ":"
)

type TokenType int

const (
	Cosmos TokenType = iota
	Erc20
	Cw20
)

func NewCw20Coin(contractAddress string, amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(CW20+TYPE_SEPARATOR+contractAddress, amount)
}

func NewErc20Coin(contractAddress string, amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(ERC20+TYPE_SEPARATOR+contractAddress, amount)
}

// ParseDenom validates and classifies a denom. Malformed denoms using the
// reserved xerc20 or xcw20 prefixes return an error instead of falling back to
// the native Cosmos bank path.
func ParseDenom(denom string) (TokenType, string, error) {
	if err := sdk.ValidateDenom(denom); err != nil {
		return Cosmos, denom, err
	}

	tokenType, contractAddress, found := strings.Cut(denom, TYPE_SEPARATOR)
	if !found {
		return Cosmos, denom, nil
	}

	switch tokenType {
	case ERC20:
		if !common.IsHexAddress(contractAddress) {
			return Cosmos, denom, fmt.Errorf("invalid ERC20 contract address %q", contractAddress)
		}
		return Erc20, contractAddress, nil
	case CW20:
		if _, err := sdk.AccAddressFromBech32(contractAddress); err != nil {
			return Cosmos, denom, fmt.Errorf("invalid CW20 contract address %q: %w", contractAddress, err)
		}
		return Cw20, contractAddress, nil
	default:
		return Cosmos, denom, nil
	}
}
