package main

import (
	"os"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	app "github.com/xpladev/xpla/app"
	"github.com/xpladev/xpla/cmd/xplad/cmd"
	"github.com/xpladev/xpla/types"
)

func main() {
	// Set address prefix and cointype
	types.SetConfig()

	rootCmd := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
