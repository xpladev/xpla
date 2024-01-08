package e2e

import (
	"context"
	"math/big"
	"strconv"

	"github.com/pkg/errors"

	ethcommon "github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	web3 "github.com/ethereum/go-ethereum/ethclient"
)

const (
	CodeOK = 0

	ErrStatusUnauthorized = "rest: unauthorized"
	ErrUnknown            = "rest: unknown error"

	HeaderContentType     = "Content-Type"
	HeaderContentTypeJson = "application/json"

	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	DELETE = "DELETE"

	REST_TIMEOUT = 2 // in sec
)

type EVMWalletInfo struct {
	CosmosWalletInfo *WalletInfo

	EthAddress    ethcommon.Address
	StringAddress string
	Nonce         uint64
}

type Method string

func NewEVMWalletInfo(mnemonics string) (*EVMWalletInfo, error) {
	cosmosWalletInfo, err := NewWalletInfo(mnemonics)
	if err != nil {
		err = errors.Wrap(err, "NewEVMWalletInfo, from NewWalletInfo")
		return nil, err
	}

	ethAddress := ethcommon.BytesToAddress(cosmosWalletInfo.PubKey.Address().Bytes())
	stringAddress := ethAddress.String()

	ret := &EVMWalletInfo{
		CosmosWalletInfo: cosmosWalletInfo,
		EthAddress:       ethAddress,
		StringAddress:    stringAddress,
	}

	return ret, nil
}

func (e *EVMWalletInfo) GetNonce(client *web3.Client) (uint64, error) {
	nonce, err := client.NonceAt(context.Background(), e.EthAddress, nil)
	if err != nil {
		err = errors.Wrap(err, "GetNonce")
		return 0, err
	}

	e.Nonce = nonce
	return nonce, nil
}

func (e *EVMWalletInfo) SendTx(client *web3.Client, to ethcommon.Address, ethAmount *big.Int, txData []byte) (ethcommon.Hash, error) {
	xplaGasPriceInt, err := strconv.ParseUint(xplaGasPrice, 10, 64)
	if err != nil {
		err = errors.Wrap(err, "SendTx, Sign")
		return ethcommon.Hash{}, err
	}

	chainId, err := client.NetworkID(context.Background())
	if err != nil {
		err = errors.Wrap(err, "SendTx, Network ID")
		return ethcommon.Hash{}, err
	}

	txStruct := &ethtypes.LegacyTx{
		Nonce:    e.Nonce,
		GasPrice: big.NewInt(int64(xplaGasPriceInt)),
		Gas:      uint64(xplaCodeGasLimit),
		Value:    big.NewInt(0),
		Data:     txData,
	}

	tx := ethtypes.NewTx(txStruct)

	ethPrivkey, _ := ethcrypto.ToECDSA(e.CosmosWalletInfo.PrivKey.Bytes())
	signedTx, err := ethtypes.SignTx(tx, ethtypes.NewEIP155Signer(chainId), ethPrivkey)
	if err != nil {
		err = errors.Wrap(err, "SendTx, Sign")
		return ethcommon.Hash{}, err
	}

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		err = errors.Wrap(err, "SendTx, SendTransaction")
		return ethcommon.Hash{}, err
	}

	return signedTx.Hash(), nil
}
