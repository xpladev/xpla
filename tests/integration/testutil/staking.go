package testutil

import (
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

// NewMsgCreateValidator test msg creator
func NewMsgCreateValidator(address sdk.ValAddress, pubKey cryptotypes.PubKey, amt sdkmath.Int) *stakingtypes.MsgCreateValidator {
	commission := stakingtypes.NewCommissionRates(sdkmath.LegacyNewDecWithPrec(10, 2), sdkmath.LegacyOneDec(), sdkmath.LegacyOneDec())
	msg, _ := stakingtypes.NewMsgCreateValidator(
		address.String(), pubKey, sdk.NewCoin(sdk.DefaultBondDenom, amt),
		stakingtypes.Description{Moniker: "NewVal"}, commission, sdkmath.OneInt(),
	)

	return msg
}

// NewMsgDelegate test msg creator
func NewMsgDelegate(delegatorAddress sdk.AccAddress, validatorAddress sdk.ValAddress, amt sdkmath.Int) *stakingtypes.MsgDelegate {

	return stakingtypes.NewMsgDelegate(
		delegatorAddress.String(), validatorAddress.String(), sdk.NewCoin(sdk.DefaultBondDenom, amt),
	)
}
