package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/xpladev/xpla/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func NewTxCmd() *cobra.Command {
	rewardTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Reward transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	rewardTxCmd.AddCommand(
		NewFundRewardPoolCmd(),
	)

	return rewardTxCmd
}

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
			amount, err := sdk.ParseCoinNormalized(args[0])
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
