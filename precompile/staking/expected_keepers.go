package staking

import (
	"context"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type StakingKeeper interface {
	Delegate(context.Context, *stakingtypes.MsgDelegate) (*stakingtypes.MsgDelegateResponse, error)
	BeginRedelegate(context.Context, *stakingtypes.MsgBeginRedelegate) (*stakingtypes.MsgBeginRedelegateResponse, error)
	Undelegate(context.Context, *stakingtypes.MsgUndelegate) (*stakingtypes.MsgUndelegateResponse, error)
}
