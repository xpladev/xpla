package integrationtest

import (
	"log"
	"os"

	xplatype "github.com/xpladev/xpla/types"
)

const (
	xplaGeneralGasLimit int64 = 240000
	xplaCodeGasLimit    int64 = 5000000
	xplaGasPrice              = "8500000000"
)

var (
	logger *log.Logger
	desc   *ServiceDesc
)

func init() {
	logger = log.New(os.Stderr, "base network", 0)

	xplatype.SetConfig()
}
