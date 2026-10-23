package multichain

import (
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type ContractArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode string          `json:"bytecode"`
}

// ExportEthWalletKey loads a test wallet's key and verifies its address.
func ExportEthWalletKey(ctx context.Context, chain *cosmos.CosmosChain, wallet ibc.Wallet) (*ecdsa.PrivateKey, error) {
	node := chain.GetFullNode()
	privateKey, _, err := node.Exec(ctx, []string{
		chain.Config().Bin, "keys", "unsafe-export-eth-key", wallet.KeyName(),
		"--keyring-backend", "test", "--home", node.HomeDir(),
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("export EVM wallet key: %w", err)
	}
	key, err := crypto.HexToECDSA(strings.TrimSpace(string(privateKey)))
	if err != nil {
		return nil, fmt.Errorf("decode EVM wallet key: %w", err)
	}
	if crypto.PubkeyToAddress(key.PublicKey) != common.BytesToAddress(wallet.Address()) {
		return nil, fmt.Errorf("exported EVM key does not match wallet address")
	}
	return key, nil
}

func DeployEVMContract(ctx context.Context, client *ethclient.Client, auth *bind.TransactOpts, artifact ContractArtifact, args ...interface{}) (common.Address, *bind.BoundContract, error) {
	parsed, err := abi.JSON(strings.NewReader(string(artifact.ABI)))
	if err != nil {
		return common.Address{}, nil, err
	}
	address, tx, contract, err := bind.DeployContract(auth, parsed, common.FromHex(artifact.Bytecode), client, args...)
	if err != nil {
		return common.Address{}, nil, err
	}
	if _, err := WaitForSuccessfulEVMReceipt(ctx, client, tx); err != nil {
		return common.Address{}, nil, err
	}
	return address, contract, nil
}

func WaitForSuccessfulEVMReceipt(ctx context.Context, client *ethclient.Client, tx *ethtypes.Transaction) (*ethtypes.Receipt, error) {
	receipt, err := bind.WaitMined(ctx, client, tx)
	if err != nil {
		return nil, fmt.Errorf("wait for EVM transaction %s: %w", tx.Hash(), err)
	}
	if receipt.Status != ethtypes.ReceiptStatusSuccessful {
		return nil, fmt.Errorf("EVM transaction %s failed with receipt status %d", tx.Hash(), receipt.Status)
	}
	return receipt, nil
}
