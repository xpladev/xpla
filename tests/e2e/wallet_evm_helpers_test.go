package e2e

import (
	"context"
	"math/big"
	"testing"

	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

func TestEVMWalletTransactor(t *testing.T) {
	key, err := ethsecp256k1.GenerateKey()
	require.NoError(t, err)
	wallet := &EVMWalletInfo{
		CosmosWalletInfo: &WalletInfo{PrivKey: key, Sequence: 7},
		EthAddress:       common.BytesToAddress(key.PubKey().Address()),
		Nonce:            11,
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	chainID := big.NewInt(47)
	auth, err := wallet.NewTransactor(ctx, chainID)
	require.NoError(t, err)
	require.Same(t, ctx, auth.Context)
	require.Equal(t, wallet.EthAddress, auth.From)
	require.Nil(t, auth.Nonce)

	transactions := []struct {
		name string
		data ethtypes.TxData
	}{
		{"legacy", &ethtypes.LegacyTx{
			Nonce: 3, GasPrice: big.NewInt(1), Gas: 21_000,
			To: &wallet.EthAddress, Value: big.NewInt(1),
		}},
		{"dynamic fee", &ethtypes.DynamicFeeTx{
			ChainID: chainID, Nonce: 3, GasTipCap: big.NewInt(1), GasFeeCap: big.NewInt(2), Gas: 21_000,
			To: &wallet.EthAddress, Value: big.NewInt(1),
		}},
	}
	for _, tc := range transactions {
		t.Run(tc.name, func(t *testing.T) {
			tx := ethtypes.NewTx(tc.data)
			signed, err := auth.Signer(auth.From, tx)
			require.NoError(t, err)
			require.Equal(t, chainID, signed.ChainId())
			require.Equal(t, uint64(3), signed.Nonce())
			sender, err := ethtypes.Sender(ethtypes.LatestSignerForChainID(chainID), signed)
			require.NoError(t, err)
			require.Equal(t, wallet.EthAddress, sender)
			_, err = auth.Signer(common.Address{}, tx)
			require.ErrorIs(t, err, bind.ErrNotAuthorized)
		})
	}
	require.Equal(t, uint64(7), wallet.CosmosWalletInfo.Sequence)
	require.Equal(t, uint64(11), wallet.Nonce)
}

func TestEVMWalletTransactorInvalidKey(t *testing.T) {
	wallet := &EVMWalletInfo{
		CosmosWalletInfo: &WalletInfo{PrivKey: &ethsecp256k1.PrivKey{Key: make([]byte, 32)}},
	}
	auth, err := wallet.NewTransactor(context.Background(), big.NewInt(47))
	require.Error(t, err)
	require.Nil(t, auth)
}
