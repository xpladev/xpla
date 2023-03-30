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

	xplaLowGasLimit             int64 = 24
	xplaGeneralGasLimit         int64 = 300000
	xplaXATPTransferGasLimit    int64 = 600000
	xplaProposalGasLimit        int64 = 500000
	xplaCreatePairGasLimit      int64 = 700000
	xplaPairGasLimit            int64 = 800000
	xplaCodeGasLimit            int64 = 6000000
	xplaPairInstantiateGasLimit int64 = 8000000

	xplaGasPrice = "8500000000"
	tknGasPrice  = "0.085"
)

var (
	logger *log.Logger
	desc   *ServiceDesc
)

func init() {
	logger = log.New(os.Stderr, "base network", 0)

	xplatype.SetConfig()
}
