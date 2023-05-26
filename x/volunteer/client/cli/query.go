package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/xpladev/xpla/x/volunteer/types"
)

func GetQueryCmd() *cobra.Command {
	volunteerValidatorQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for volunteer module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 1,
		RunE:                       client.ValidateCmd,
	}

	volunteerValidatorQueryCmd.AddCommand(
		GetCmdQueryVolunteerValidators(),
	)

	return volunteerValidatorQueryCmd
}

func GetCmdQueryVolunteerValidators() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validators",
		Args:  cobra.NoArgs,
		Short: "Query for all volunteer validators",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query list about all volunteer validator on a network.
			
			Example:
			$ %s query volunteer validators`, version.AppName),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}

			queryClient := types.NewQueryClient(clientCtx)
			result, err := queryClient.VolunteerValidators(cmd.Context(), &types.QueryVolunteerValidatorsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}
