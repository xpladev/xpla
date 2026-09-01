package types_test

import (
	"os"
	"testing"

	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/bank/types"
)

func TestMain(m *testing.M) {
	xplatypes.SetConfig()
	os.Exit(m.Run())
}

func TestParseDenom(t *testing.T) {
	tests := []struct {
		input         string
		expectedType  types.TokenType
		expectedDenom string
		expectError   bool
	}{
		{"xerc20:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", types.Erc20, "A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", false},
		{"xerc20:0xA2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", types.Erc20, "0xA2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", false},
		{"xcw20:xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4", types.Cw20, "xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4", false},
		{"xerc20foo:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", types.Cosmos, "xerc20foo:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546", false},
		{"xcw20-old:xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4", types.Cosmos, "xcw20-old:xpla1hz3svgdhmv67lsqlduu0tcnd3f75c0xr0mu48l6ywuwlz43zssjqc0z2h4", false},
		{"uatom", types.Cosmos, "uatom", false},
		{"ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5", types.Cosmos, "ibc/8E27BA2D5493AF5636760E354E46004562C46AB7EC0CC4C1CA14E9E20E2545B5", false},
		{"aphoton", types.Cosmos, "aphoton", false},
		{"xerc20:invalid", types.Cosmos, "xerc20:invalid", true},
		{"xerc20:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546:extra", types.Cosmos, "xerc20:A2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546:extra", true},
		{"xcw20:xpla1invalid", types.Cosmos, "xcw20:xpla1invalid", true},
		{"?", types.Cosmos, "?", true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokenType, denom, err := types.ParseDenom(tt.input)
			if tt.expectError {
				if err == nil {
					t.Fatalf("ParseDenom(%s) expected an error", tt.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDenom(%s) unexpected error: %v", tt.input, err)
			}
			if tokenType != tt.expectedType || denom != tt.expectedDenom {
				t.Errorf("ParseDenom(%s) = (%v, %s); want (%v, %s)", tt.input, tokenType, denom, tt.expectedType, tt.expectedDenom)
			}
		})
	}
}
