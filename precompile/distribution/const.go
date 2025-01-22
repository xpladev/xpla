package distribution

const (
	hexAddress = "0x1000000000000000000000000000000000000003"
	abiFile    = "IDistribution.abi"
)

type MethodDistribution string

const (
	WithdrawDelegatorReward MethodDistribution = "withdrawDelegatorReward"
)
