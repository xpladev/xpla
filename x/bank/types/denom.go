package types

import (
	"strings"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	ERC20 = "erc20"
	CW20  = "cw20"
)

type TokenType int

const (
	Cosmos TokenType = iota
	Erc20
	Cw20
)

func NewCw20Coin(contractAddress string, amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(CW20+"/"+contractAddress, amount)
}

func NewErc20Coin(contractAddress string, amount sdkmath.Int) sdk.Coin {
	return sdk.NewCoin(ERC20+"/"+contractAddress, amount)
}

func ParseDenom(denom string) (TokenType, string) {
	res := strings.Split(denom, "/")

	if len(res) == 2 {
		if strings.HasPrefix(denom, ERC20) {
			return Erc20, res[1]
		}

		if strings.HasPrefix(denom, CW20) {
			return Cw20, res[1]
		}
	}

	return Cosmos, denom
}
