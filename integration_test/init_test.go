package integrationtest

import (
	"log"
	"os"

	xplatype "github.com/xpladev/xpla/types"
)

const (
	xplaLowGasLimit             int64 = 24
	xplaGeneralGasLimit         int64 = 240000
	xplaXATPTransferGasLimit    int64 = 600000
	xplaProposalGasLimit        int64 = 500000
	xplaCreatePairGasLimit      int64 = 650000
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
