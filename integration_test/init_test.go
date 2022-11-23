package integrationtest

import (
	"log"
	"os"

	xplatype "github.com/xpladev/xpla/types"
)

const (
	xplaGasLimit int64 = 240000
	xplaGasPrice       = "850000000000"
)

var (
	logger *log.Logger
	desc   *ServiceDesc
)

func init() {
	logger = log.New(os.Stderr, "base network", 0)

	xplatype.SetConfig()
}
