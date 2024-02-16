package testutil

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// NewMsgCreateValidator test msg creator
func NewMsgCreateValidator(address sdk.ValAddress, pubKey cryptotypes.PubKey, amt sdk.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(sdk.NewDecWithPrec(10, 2), sdk.OneDec(), sdk.OneDec())
	msg, _ := stakingtypes.NewMsgCreateValidator(
		address, pubKey, sdk.NewCoin(sdk.DefaultBondDenom, amt),
		stakingtypes.Description{}, commission, sdk.OneInt(),
	)

	return msg
}

// NewMsgDelegate test msg creator
func NewMsgDelegate(delegatorAddress sdk.AccAddress, validatorAddress sdk.ValAddress, amt sdk.Int) *stakingtypes.MsgDelegate {

	return stakingtypes.NewMsgDelegate(
		delegatorAddress, validatorAddress, sdk.NewCoin(sdk.DefaultBondDenom, amt),
	)
}
