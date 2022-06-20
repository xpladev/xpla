package main

import (
	"os"

	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	app "github.com/c2xdev/noname/v1/app"
	"github.com/c2xdev/noname/v1/cmd/nonamed/cmd"
	"github.com/c2xdev/noname/v1/types"
)

func main() {
	// Set address prefix and cointype
	types.SetConfig()

	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}
