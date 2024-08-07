package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/xpladev/xpla/x/reward/types"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
)

const (
	MaxMessagesPerTxDefault = 0
)

// NewTxCmd returns a root CLI command handler for all x/reward transaction commands.
func NewTxCmd() *cobra.Command {
	distTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Reward transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	distTxCmd.AddCommand(
		NewFundRewardPoolCmd(),
	)

	return distTxCmd
}

type newGenerateOrBroadcastFunc func(client.Context, *pflag.FlagSet, ...sdk.Msg) error

func NewFundRewardPoolCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fund-reward-pool [amount]",
		Args:  cobra.ExactArgs(1),
		Short: "Funds the reward pool with the specified amount",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Funds the reward pool with the specified amount

Example:
$ %s tx reward fund-reward-pool 100axpla --from mykey
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			depositorAddr := clientCtx.GetFromAddress()
			amount, err := sdk.ParseCoinsNormalized(args[0])
			if err != nil {
				return err
			}

			msg := types.NewMsgFundRewardPool(amount, depositorAddr)

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
