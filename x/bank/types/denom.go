package types

import "strings"

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
