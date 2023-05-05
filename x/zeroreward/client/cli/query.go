package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/xpladev/xpla/x/zeroreward/types"
)

func GetQueryCmd() *cobra.Command {
	zeroRewardValidatorQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for zeroreward module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 1,
		RunE:                       client.ValidateCmd,
	}

	zeroRewardValidatorQueryCmd.AddCommand(
		GetCmdQueryZeroRewardValidators(),
	)

	return zeroRewardValidatorQueryCmd
}

func GetCmdQueryZeroRewardValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Args:  cobra.NoArgs,
		Short: "Query for all zero reward validators",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query list about all zero reward validator on a network.
			
			Example:
			$ %s query zeroreward validators`, version.AppName),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			result, err := queryClient.ZeroRewardValidators(cmd.Context(), &types.QueryZeroRewardValidatorsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
