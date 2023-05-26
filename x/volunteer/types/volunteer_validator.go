package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewVolunteerValidator(valAddress sdk.ValAddress, power int64) VolunteerValidator {
	return VolunteerValidator{
		Address:    valAddress.String(),
		Power:      power,
		IsDeleting: false,
	}
}

func (zv *VolunteerValidator) Delete() {
	zv.IsDeleting = true
}
