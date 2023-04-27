package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingcli "github.com/cosmos/cosmos-sdk/x/staking/client/cli"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func GetSubmitProposalRegisterSpecialValidator() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-special-validator [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a register special validator proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a register special validator proposal along with an initial deposit.
			The proposal details must be supplied via a JSON file.
			
			Example:
$ %s tx gov submit-proposal register-special-validator <path/to/proposal.json>

Where proposal.json contains:

{
	"title": "Register special validator proposal",
	"description": "Registration of validators independent of the active set",
	"deposit": "10000000%s"
}
`, version.AppName, xplatypes.DefaultDenom)),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := ParseRegisterSpecialValidatorProposalWithDeposit(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			fs := cmd.Flags()

			strAmount, _ := fs.GetString(stakingcli.FlagAmount)
			amount, err := sdk.ParseCoinNormalized(strAmount)
			if err != nil {
				return err
			}

			from := clientCtx.GetFromAddress()

			var pubKey cryptotypes.PubKey
			pkStr, err := fs.GetString(stakingcli.FlagPubKey)
			if err != nil {
				return err
			}
			if err := clientCtx.Codec.UnmarshalInterfaceJSON([]byte(pkStr), &pubKey); err != nil {
				return err
			}

			moniker, _ := fs.GetString(stakingcli.FlagMoniker)
			identity, _ := fs.GetString(stakingcli.FlagIdentity)
			website, _ := fs.GetString(stakingcli.FlagWebsite)
			security, _ := fs.GetString(stakingcli.FlagSecurityContact)
			details, _ := fs.GetString(stakingcli.FlagDetails)
			description := stakingtypes.NewDescription(
				moniker,
				identity,
				website,
				security,
				details,
			)

			content, err := types.NewRegisterSpecialValidatorProposal(proposal.Title, proposal.Description, from, sdk.ValAddress(from), pubKey, amount, description)
			if err != nil {
				return err
			}

			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	cmd.Flags().AddFlagSet(stakingcli.FlagSetPublicKey())
	cmd.Flags().AddFlagSet(stakingcli.FlagSetAmount())
	cmd.Flags().AddFlagSet(flagSetDescriptionCreate())

	cmd.Flags().String(stakingcli.FlagIP, "", fmt.Sprintf("The node's public IP. It takes effect only when used in combination with --%s", flags.FlagGenerateOnly))
	cmd.Flags().String(stakingcli.FlagNodeID, "", "The node's ID")

	_ = cmd.MarkFlagRequired(flags.FlagFrom)
	_ = cmd.MarkFlagRequired(stakingcli.FlagAmount)
	_ = cmd.MarkFlagRequired(stakingcli.FlagPubKey)
	_ = cmd.MarkFlagRequired(stakingcli.FlagMoniker)

	return cmd
}

func GetSubmitProposalUnregisterSpecialValidator() *cobra.Command {
	bech32PrefixAccAddr := sdk.GetConfig().GetBech32AccountAddrPrefix()

	cmd := &cobra.Command{
		Use:   "unregister-special-validator [proposal-file]",
		Args:  cobra.ExactArgs(1),
		Short: "Submit a unregister special validator proposal",
		Long: strings.TrimSpace(
			fmt.Sprintf(`Submit a unregister special validator proposal along with an initial deposit.
			The proposal details must be supplied via a JSON file.
			
			Example:
$ %s tx gov submit-proposal unregister-special-validator <path/to/proposal.json>

Where proposal.json contains:

{
	"title": "Unregister special validator proposal",
	"description": "Unregistration of validators independent of the active set",
	"validator_address": "%svaloper1luqjvjyns9e92h06tq6zqtw76k8xtegffyqca7",
	"deposit": "10000000%s"
}
`, version.AppName, bech32PrefixAccAddr, xplatypes.DefaultDenom)),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			proposal, err := ParseUnregisterSpecialValidatorProposalWithDeposit(clientCtx.Codec, args[0])
			if err != nil {
				return err
			}

			deposit, err := sdk.ParseCoinsNormalized(proposal.Deposit)
			if err != nil {
				return err
			}

			valAddr, err := sdk.ValAddressFromBech32(proposal.ValidatorAddress)
			if err != nil {
				return err
			}

			content := types.NewUnregisterSpecialValidatorProposal(proposal.Title, proposal.Description, valAddr)

			from := clientCtx.GetFromAddress()
			msg, err := govtypes.NewMsgSubmitProposal(content, deposit, from)
			if err != nil {
				return err
			}

			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	_ = cmd.MarkFlagRequired(flags.FlagFrom)

	return cmd
}
