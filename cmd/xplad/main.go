package main

import (
	"os"

	sdkmath "cosmossdk.io/math"

	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	sdk "github.com/cosmos/cosmos-sdk/types"

	app "github.com/xpladev/xpla/app"
	"github.com/xpladev/xpla/cmd/xplad/cmd"
	"github.com/xpladev/xpla/types"
)

func main() {
	// Set address prefix and cointype
	types.SetConfig()

	err := sdk.RegisterDenom(types.DefaultDenom, sdkmath.LegacyNewDecWithPrec(1, types.DefaultDenomPrecision))
	if err != nil {
		panic(err)
	}

	rootCmd := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		os.Exit(1)
	}
}
