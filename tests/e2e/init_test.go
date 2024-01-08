package e2e

import (
	"log"
	"os"

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
	logger *log.Logger
	desc   *ServiceDesc
)

func init() {
	logger = log.New(os.Stderr, "base network", 0)

	xplatype.SetConfig()
}
