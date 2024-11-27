package staking

const (
	hexAddress = "0x1000000000000000000000000000000000000002"
	abiFile    = "IStaking.abi"
)

type MethodStaking string

const (
	Delegate        MethodStaking = "delegate"
	BeginRedelegate MethodStaking = "beginRedelegate"
	Undelegate      MethodStaking = "undelegate"
)
