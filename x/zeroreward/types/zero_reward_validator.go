package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewZeroRewardValidator(valAddress sdk.ValAddress, power int64) ZeroRewardValidator {
	return ZeroRewardValidator{
		Address:    valAddress.String(),
		Power:      power,
		IsDeleting: false,
	}
}

func (zv *ZeroRewardValidator) Delete() {
	zv.IsDeleting = true
}
