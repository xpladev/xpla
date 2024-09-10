package types

import (
	"context"
	"time"

	"cosmossdk.io/core/address"
	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	GetValidator(ctx context.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, err error)
	GetValidatorByConsAddr(ctx context.Context, consAddr sdk.ConsAddress) (validator stakingtypes.Validator, found bool)
	BondDenom(ctx context.Context) (res string)
	SetValidator(ctx context.Context, validator stakingtypes.Validator)
	SetValidatorByConsAddr(ctx context.Context, validator stakingtypes.Validator) error
	SetNewValidatorByPowerIndex(ctx context.Context, validator stakingtypes.Validator)
	Delegate(
		ctx context.Context, delAddr sdk.AccAddress, bondAmt sdkmath.Int, tokenSrc stakingtypes.BondStatus,
		validator stakingtypes.Validator, subtractAccount bool,
	) (newShares sdkmath.LegacyDec, err error)
	Undelegate(
		ctx context.Context, delAddr sdk.AccAddress, valAddr sdk.ValAddress, sharesAmount sdkmath.LegacyDec,
	) (time.Time, error)
	Hooks() stakingtypes.StakingHooks
	ValidatorAddressCodec() address.Codec
}

type DistributionKeeper interface {
	WithdrawValidatorCommission(ctx context.Context, valAddr sdk.ValAddress) (sdk.Coins, error)
	FundCommunityPool(ctx context.Context, amount sdk.Coins, sender sdk.AccAddress) error
}
