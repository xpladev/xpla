package cli

import (
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/spf13/cobra"
	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

func GetSubmitProposalRegisterXatp() *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

	cmd := &cobra.Command{
		Use:   "register-xatp [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a register xatp proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a register xatp proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.

Example:
$ %s tx gov submit-proposal register-xatp <path/to/proposal.json> --from=<key_or_address>

Where proposal.json contains:

{
	"title": "Register proposal",
	"description": "To replace fee with CTXT",
	"xatp": {
		"denom": "CTXT",
		"token": "%s1r57m20afwdhkwy67520p8vzdchzecesmlmc8k8w2z7t3h9aevjvs35x4r5",
		"pair": "%s1sdzaas0068n42xk8ndm6959gpu6n09tajmeuq7vak8t9qt5jrp6szltsnk",
		"decimals": 6
	},
	"deposit": "1000%s"
}
`, version.AppName, bech32PrefixAccAddr, bech32PrefixAccAddr, xplatypes.DefaultDenom),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := ParseRegisterXatpProposalWithDeposit(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			content := types.NewRegisterXatpProposal(proposal.Title, proposal.Description, proposal.Xatp.Token, proposal.Xatp.Pair, proposal.Xatp.Denom, int(proposal.Xatp.Decimals))

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

		},
	}

	return cmd
}

func GetSubmitProposalUnregisterXatp() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unregister-xatp [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a unregister xatp proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a unregister xatp proposal along with an initial deposit.
The proposal details must be supplied via a JSON file.

Example:
$ %s tx gov submit-proposal unregister-xatp <path/to/proposal.json> --from=<key_or_address>

Where proposal.json contains:

{
	"title": "Unregister proposal",
	"description": "Making it impossible to replace the fee with CTXT",
	"denom": "CTXT",
	"deposit": "1000%s"
}
`, version.AppName, xplatypes.DefaultDenom),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := ParseUnregisterXatpProposalWithDeposit(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			content := types.NewUnregisterXatpProposal(proposal.Title, proposal.Description, proposal.Denom)

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)

		},
	}

	return cmd
}
