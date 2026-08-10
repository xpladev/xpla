package types

import (
	"bytes"
	"math"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestDefaultParams(t *testing.T) {
	p := DefaultParams()
	require.NoError(t, p.Validate())
	require.True(t, p.Enabled)
	require.True(t, p.AllocationRate.Equal(sdkmath.LegacyNewDecWithPrec(20, 2)))
	require.Equal(t, uint64(100_000), p.SettlementIntervalBlocks)
	require.Equal(t, "10000000000000000000000", p.MinFeeAmount.Amount.String())
	require.Equal(t, "500000000000000000000000", p.MaxFeeAmount.Amount.String())
}

func TestParamsValidate(t *testing.T) {
	negative := sdkmath.NewInt(-1)
	tests := []struct {
		name   string
		mutate func(*Params)
	}{
		{"nil rate", func(p *Params) { p.AllocationRate = sdkmath.LegacyDec{} }},
		{"negative rate", func(p *Params) { p.AllocationRate = sdkmath.LegacyMustNewDecFromStr("-0.1") }},
		{"rate above one", func(p *Params) { p.AllocationRate = sdkmath.LegacyMustNewDecFromStr("1.000000000000000001") }},
		{"zero interval", func(p *Params) { p.SettlementIntervalBlocks = 0 }},
		{"interval above int64", func(p *Params) { p.SettlementIntervalBlocks = uint64(math.MaxInt64) + 1 }},
		{"negative min", func(p *Params) { p.MinFeeAmount = sdk.Coin{Denom: TargetDenom, Amount: negative} }},
		{"nil min", func(p *Params) { p.MinFeeAmount = sdk.Coin{Denom: TargetDenom} }},
		{"wrong min denom", func(p *Params) { p.MinFeeAmount.Denom = "ufoo" }},
		{"wrong max denom", func(p *Params) { p.MaxFeeAmount.Denom = "ufoo" }},
		{"max equals min", func(p *Params) { p.MaxFeeAmount = p.MinFeeAmount }},
		{"max below min", func(p *Params) { p.MaxFeeAmount.Amount = p.MinFeeAmount.Amount.SubRaw(1) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := DefaultParams()
			tt.mutate(&p)
			require.Error(t, p.Validate())
		})
	}
}

func TestParamsValueSnapshot(t *testing.T) {
	p := DefaultParams()
	active := p
	p.Enabled = false
	p.AllocationRate = sdkmath.LegacyOneDec()
	p.SettlementIntervalBlocks = 1
	require.True(t, active.Enabled)
	require.True(t, active.AllocationRate.Equal(DefaultAllocationRate))
	require.Equal(t, uint64(100_000), active.SettlementIntervalBlocks)
}

func TestParamsValidationRejectsEmptyProtobufDecimal(t *testing.T) {
	var params Params
	require.NoError(t, params.Unmarshal([]byte{0x12, 0x00}))
	require.Error(t, params.Validate())
}

func TestParamsProtobufRejectsMalformedCoinAmount(t *testing.T) {
	params := DefaultParams()
	encoded, err := params.Marshal()
	require.NoError(t, err)

	validAmount := []byte(params.MinFeeAmount.Amount.String())
	malformedAmount := bytes.Repeat([]byte("x"), len(validAmount))
	encoded = bytes.Replace(encoded, validAmount, malformedAmount, 1)

	var decoded Params
	require.Error(t, decoded.Unmarshal(encoded))
}
