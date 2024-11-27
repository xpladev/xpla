package distribution

import (
	"context"

	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
)

type DistributionKeeper interface {
	WithdrawDelegatorReward(ctx context.Context, msg *distributiontypes.MsgWithdrawDelegatorReward) (*distributiontypes.MsgWithdrawDelegatorRewardResponse, error)
}
