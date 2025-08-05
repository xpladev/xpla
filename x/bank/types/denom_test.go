package types_test

import (
	"testing"

	"github.com/xpladev/xpla/x/bank/types"
)

func TestParseDenom(t *testing.T) {
	tests := []struct {
		input         string
		expectedType  types.TokenType
		expectedDenom string
	}{
		{"xerc20:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", types.Erc20, "A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546"},
		{"xcw20:xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4", types.Cw20, "xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4"},
		{"uatom", types.Cosmos, "uatom"},
		{"ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5", types.Cosmos, "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5"},
		{"aphoton", types.Cosmos, "aphoton"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenType, denom := types.ParseDenom(tt.input)
			if tokenType != tt.expectedType || denom != tt.expectedDenom {
				t.Errorf("ParseDenom(%s) = (%v, %s); want (%v, %s)", tt.input, tokenType, denom, tt.expectedType, tt.expectedDenom)
			}
		})
	}
}
