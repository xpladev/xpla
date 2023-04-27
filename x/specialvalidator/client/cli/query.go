package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func GetQueryCmd() *cobra.Command {
	specialValidatorQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for specialvalidator module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 1,
		RunE:                       client.ValidateCmd,
	}

	specialValidatorQueryCmd.AddCommand(
		GetCmdQuerySpecialValidators(),
	)

	return specialValidatorQueryCmd
}

func GetCmdQuerySpecialValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "special-validators",
		Args:  cobra.NoArgs,
		Short: "Query for all special validators",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query list about all special validator on a network.
			
			Example:
			$ %s query specialvalidato special-validators`, version.AppName),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			result, err := queryClient.Specialvalidators(cmd.Context(), &types.QuerySpecialValidatorsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
