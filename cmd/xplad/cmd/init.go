package cmd

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	cmtcfg "github.com/cometbft/cometbft/config"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/cosmos/go-bip39"
	"github.com/spf13/cobra"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math/unsafe"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/input"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/server"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	genutilcli "github.com/cosmos/cosmos-sdk/x/genutil/client/cli"
	genutiltypes "github.com/cosmos/cosmos-sdk/x/genutil/types"

	xpla "github.com/xpladev/xpla/app"
	xplatypes "github.com/xpladev/xpla/types"
)

type initPrintInfo struct {
	Moniker    string          `json:"moniker"`
	ChainID    string          `json:"chain_id"`
	NodeID     string          `json:"node_id"`
	AppMessage json.RawMessage `json:"app_message"`
}

// InitCmd runs init using the app's DefaultGenesis so XPLA-specific default
// genesis values are included in generated genesis.json.
func InitCmd(app *xpla.XplaApp, defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [moniker]",
		Short: "Initialize private validator, p2p, genesis, and application configuration files",
		Long:  "Initialize validators's and node's configuration files.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)
			serverCtx := server.GetServerContextFromCmd(cmd)
			cfg := serverCtx.Config
			cfg.SetRoot(clientCtx.HomeDir)

			chainID, _ := cmd.Flags().GetString(flags.FlagChainID)
			switch {
			case chainID != "":
			case clientCtx.ChainID != "":
				chainID = clientCtx.ChainID
			default:
				chainID = fmt.Sprintf("test-chain-%v", unsafe.Str(6))
			}

			var mnemonic string
			recoverFlag, _ := cmd.Flags().GetBool(genutilcli.FlagRecover)
			if recoverFlag {
				inBuf := bufio.NewReader(cmd.InOrStdin())
				value, err := input.GetString("Enter your bip39 mnemonic", inBuf)
				if err != nil {
					return err
				}
				mnemonic = value
				if !bip39.IsMnemonicValid(mnemonic) {
					return errors.New("invalid mnemonic")
				}
			}

			initHeight, _ := cmd.Flags().GetInt64(flags.FlagInitHeight)
			if initHeight < 1 {
				initHeight = 1
			}

			defaultDenom, _ := cmd.Flags().GetString(genutilcli.FlagDefaultBondDenom)
			if defaultDenom != "" && defaultDenom != xplatypes.DefaultDenom {
				return fmt.Errorf("unsupported default denom %q: XPLA genesis must use %q", defaultDenom, xplatypes.DefaultDenom)
			}

			nodeID, _, err := genutil.InitializeNodeValidatorFilesFromMnemonic(cfg, mnemonic)
			if err != nil {
				return err
			}
			cfg.Moniker = args[0]

			genFile := cfg.GenesisFile()
			overwrite, _ := cmd.Flags().GetBool(genutilcli.FlagOverwrite)

			if _, err = os.Stat(genFile); err != nil {
				if !os.IsNotExist(err) {
					return err
				}
			} else if !overwrite {
				return fmt.Errorf("genesis.json file already exists: %v", genFile)
			}

			appGenState := app.DefaultGenesis()
			appState, err := json.MarshalIndent(appGenState, "", " ")
			if err != nil {
				return errorsmod.Wrap(err, "failed to marshal default genesis state")
			}

			appGenesis := &genutiltypes.AppGenesis{}
			if _, err := os.Stat(genFile); err == nil {
				appGenesis, err = genutiltypes.AppGenesisFromFile(genFile)
				if err != nil {
					return errorsmod.Wrap(err, "failed to read genesis doc from file")
				}
			}
			appGenesis.AppName = version.AppName
			appGenesis.AppVersion = version.Version
			appGenesis.ChainID = chainID
			appGenesis.AppState = appState
			appGenesis.InitialHeight = initHeight
			appGenesis.Consensus = &genutiltypes.ConsensusGenesis{
				Validators: nil,
				Params:     cmttypes.DefaultConsensusParams(),
			}

			consensusKey, err := cmd.Flags().GetString(genutilcli.FlagConsensusKeyAlgo)
			if err != nil {
				return errorsmod.Wrap(err, "failed to get consensus key algo")
			}
			appGenesis.Consensus.Params.Validator.PubKeyTypes = []string{consensusKey}

			if err = genutil.ExportGenesisFile(appGenesis, genFile); err != nil {
				return errorsmod.Wrap(err, "failed to export genesis file")
			}

			toPrint := initPrintInfo{
				Moniker:    cfg.Moniker,
				ChainID:    chainID,
				NodeID:     nodeID,
				AppMessage: appState,
			}
			out, err := json.MarshalIndent(toPrint, "", " ")
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(os.Stderr, "%s\n", out); err != nil {
				return err
			}

			cfg.Mempool.Type = cmtcfg.MempoolTypeApp
			cmtcfg.WriteConfigFile(filepath.Join(cfg.RootDir, "config", "config.toml"), cfg)

			otelFile := filepath.Join(clientCtx.HomeDir, "config", telemetry.OtelFileName)
			if err := os.WriteFile(otelFile, []byte{}, 0o600); err != nil {
				return errorsmod.Wrap(err, "failed to create otel.yaml file")
			}

			return nil
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "node's home directory")
	cmd.Flags().BoolP(genutilcli.FlagOverwrite, "o", false, "overwrite the genesis.json file")
	cmd.Flags().Bool(genutilcli.FlagRecover, false, "provide seed phrase to recover existing key instead of creating")
	cmd.Flags().String(flags.FlagChainID, "", "genesis file chain-id, if left blank will be randomly created")
	cmd.Flags().String(genutilcli.FlagDefaultBondDenom, xplatypes.DefaultDenom, "genesis file default denomination")
	cmd.Flags().Int64(flags.FlagInitHeight, 1, "specify the initial block height at genesis")
	cmd.Flags().String(genutilcli.FlagConsensusKeyAlgo, ed25519.KeyType, "algorithm to use for the consensus key")

	return cmd
}
