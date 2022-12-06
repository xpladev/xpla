package cli

import (
	"encoding/hex"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"github.com/xpladev/xpla/x/proxyevm/types"
)

const (
	flagAmount = "amount"
)

// NewTxCmd returns a root CLI command handler for all x/proxyevm transaction commands.
func NewTxCmd() *cobra.Command {
	distTxCmd := &cobra.Command{
		Use:                        types.ModuleName,
		Short:                      "Proxy evm transactions subcommands",
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	distTxCmd.AddCommand(
		NewCallEVM(),
	)

	return distTxCmd
}

func NewCallEVM() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "call [hex_contract_addr] [hex_encoded_data] --amount [coins,optional]",
		Args:    cobra.ExactArgs(2),
		Aliases: []string{"run", "execute", "exec", "ex", "c"},
		Short:   "Call a command on a evm contract",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			msg, err := parseCallEVMArgs(args[0], args[1], clientCtx.GetFromAddress(), cmd.Flags())
			if err != nil {
				return err
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String(flagAmount, "", "Coins to send to the contract along with command")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func parseCallEVMArgs(contractAddr string, data string, sender sdk.AccAddress, flags *flag.FlagSet) (types.MsgCallEVM, error) {
	amountStr, err := flags.GetString(flagAmount)
	if err != nil {
		return types.MsgCallEVM{}, fmt.Errorf("amount: %s", err)
	}

	amount, err := sdk.ParseCoinsNormalized(amountStr)
	if err != nil {
		return types.MsgCallEVM{}, err
	}

	hexData, err := hex.DecodeString(data)
	if err != nil {
		return types.MsgCallEVM{}, err
	}

	return types.MsgCallEVM{
		Sender:   sender.String(),
		Contract: contractAddr,
		Data:     hexData,
		Funds:    amount,
	}, nil
}
