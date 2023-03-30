package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/spf13/cobra"
	"github.com/xpladev/xpla/x/xatp/types"
)

func GetQueryCmd() *cobra.Command {
	xatpQueryCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Querying commands for xatp module",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	xatpQueryCmd.AddCommand(
		GetCmdQueryParams(),
		GetCmdQueryXatps(),
		GetCmdQueryXatp(),
		GetCmdQueryXatpPool(),
	)

	return xatpQueryCmd
}

// GetCmdQueryParams implements the query params command.
func GetCmdQueryParams() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "params",
		Args:  cobra.NoArgs,
		Short: "Query xatp params",
		RunE: func(cmd *cobra.Command, _ []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.Params(cmd.Context(), &types.QueryParamsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Params)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}

// GetCmdQueryXatps implements the query all xatps command.
func GetCmdQueryXatps() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "xatps",
		Short: "Query for all xatps",
		Args:  cobra.NoArgs,
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query details about all xatps on a network.

Example:
$ %s query xatp xatps
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)
			result, err := queryClient.Xatps(cmd.Context(), &types.QueryXatpsRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(result)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryXatp implements the xatp query command.
func GetCmdQueryXatp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "xatp [denom]",
		Short: "Query a xatp",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query details about an individual xatp.

Example:
$ %s query xatp xatp CTXT
`,
				version.AppName,
			),
		),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			params := &types.QueryXatpRequest{Denom: args[0]}
			res, err := queryClient.Xatp(cmd.Context(), params)
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(&res.Xatp)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)

	return cmd
}

// GetCmdQueryXatpPool returns the command for fetching xatp pool info.
func GetCmdQueryXatpPool() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pool",
		Args:  cobra.NoArgs,
		Short: "Query the amount of coins in the xatp pool",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Query all coins in the xatp pool.

Example:
$ %s query xatp pool
`,
				version.AppName,
			),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientQueryContext(cmd)
			if err != nil {
				return err
			}
			queryClient := types.NewQueryClient(clientCtx)

			res, err := queryClient.XatpPool(cmd.Context(), &types.QueryXatpPoolRequest{})
			if err != nil {
				return err
			}

			return clientCtx.PrintProto(res)
		},
	}

	flags.AddQueryFlagsToCmd(cmd)
	return cmd
}
