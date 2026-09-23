package e2e

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type EVMWalletInfo struct {
	CosmosWalletInfo *WalletInfo

	EthAddress    common.Address
	StringAddress string
	Nonce         uint64
}

// NewTransactor creates a contract signer using the wallet's existing key.
// Nonce and gas options remain unset so callers can use RPC defaults or set them.
func (e *EVMWalletInfo) NewTransactor(ctx context.Context, chainID *big.Int) (*bind.TransactOpts, error) {
	key, err := crypto.ToECDSA(e.CosmosWalletInfo.PrivKey.Bytes())
	if err != nil {
		return nil, err
	}
	auth, err := bind.NewKeyedTransactorWithChainID(key, chainID)
	if err != nil {
		return nil, err
	}
	auth.Context = ctx
	return auth, nil
}
