package e2e

import (
	"log"
	"os"

	codec "github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	stakingtype "github.com/cosmos/cosmos-sdk/x/staking/types"
	xplatype "github.com/xpladev/xpla/types"
)

const (
	blocktime             = 2
	proposalBlocks        = 2
	jailBlocks            = 5
	validatorActiveBlocks = 3
	downtimeJailDuration  = 20

	xplaGeneralGasLimit  int64 = 300000
	xplaCodeGasLimit     int64 = 5000000
	xplaProposalGasLimit int64 = 500000
	xplaGasPrice               = "8500000000"
)

var (
	logger    *log.Logger
	desc      *ServiceDesc
	marshaler *codec.ProtoCodec
)

func init() {
	logger = log.New(os.Stderr, "base network", 0)

	xplatype.SetConfig()

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	stakingtype.RegisterInterfaces(interfaceRegistry)
	marshaler = codec.NewProtoCodec(interfaceRegistry)
}
