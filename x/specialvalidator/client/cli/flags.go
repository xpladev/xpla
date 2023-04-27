package cli

import (
	flag "github.com/spf13/pflag"

	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"
)

func flagSetDescriptionCreate() *flag.FlagSet {
	fs := flag.NewFlagSet("", flag.ContinueOnError)

	fs.String(stakingcli.FlagMoniker, "", "The validator's name")
	fs.String(stakingcli.FlagIdentity, "", "The optional identity signature (ex. UPort or Keybase)")
	fs.String(stakingcli.FlagWebsite, "", "The validator's (optional) website")
	fs.String(stakingcli.FlagSecurityContact, "", "The validator's (optional) security contact email")
	fs.String(stakingcli.FlagDetails, "", "The validator's (optional) details")

	return fs
}
