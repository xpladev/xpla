package integrationtest

import (
	"context"
	"encoding/json"
	"os"
	"sync"

	"github.com/pkg/errors"
	"google.golang.org/grpc"

	cosmwasmtype "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/simapp"
	simappparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdktype "github.com/cosmos/cosmos-sdk/types"
	txtype "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	xauthsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ethhd "github.com/evmos/ethermint/crypto/hd"
	evmtypes "github.com/evmos/ethermint/x/evm/types"

	xplatypes "github.com/xpladev/xpla/types"
)

const (
	Prefix  = "xpla"
	ChainID = "localtest_1-1"
)

type WalletInfo struct {
	sync.Mutex

	IsSrc         bool
	ChainId       string
	Prefix        string
	StringAddress string
	ByteAddress   sdktype.AccAddress
	PrivKey       cryptotypes.PrivKey
	PubKey        cryptotypes.PubKey
	AccountNumber uint64
	Sequence      uint64
	EncCfg        simappparams.EncodingConfig
}

func NewWalletInfo(mnemonics string) (*WalletInfo, error) {
	// derive key
	fullFundraiserPath := "m/44'/60'/0'/0/0"

	var privKey cryptotypes.PrivKey
	var pubKey cryptotypes.PubKey
	var byteAddress sdktype.AccAddress
	var stringAddress string

	devFunc := ethhd.EthSecp256k1.Derive()
	privBytes, err := devFunc(mnemonics, "", fullFundraiserPath)
	if err != nil {
		err = errors.Wrap(err, "NewWalletInfo, derive mnemonics -> rootkey")
		return nil, err
	}

	genFunc := ethhd.EthSecp256k1.Generate()
	privKey = genFunc(privBytes)
	if err != nil {
		err = errors.Wrap(err, "NewWalletInfo, rootkey -> privkey")
		return nil, err
	}
	pubKey = privKey.PubKey()

	byteAddress = sdktype.AccAddress(pubKey.Address())
	stringAddress, err = sdktype.Bech32ifyAddressBytes(Prefix, pubKey.Address())
	if err != nil {
		err = errors.Wrap(err, "NewWalletInfo, create bech32 address from byte")
		return nil, err
	}

	encCfg := simapp.MakeTestEncodingConfig()

	accountNumber, seq, err := GetAccountNumber(desc.ServiceConn, ChainID, stringAddress)
	if err != nil {
		err = errors.Wrap(err, "NewWalletInfo, get account info")
		return nil, err
	}

	ret := &WalletInfo{
		ChainId:       ChainID,
		Prefix:        Prefix,
		ByteAddress:   byteAddress,
		StringAddress: stringAddress,
		PrivKey:       privKey,
		PubKey:        pubKey,
		AccountNumber: accountNumber,
		Sequence:      seq,
		EncCfg:        encCfg,
	}

	return ret, nil
}

func (w *WalletInfo) SendTx(chainId string, msg sdktype.Msg, fee sdktype.Coin, gasLimit int64, isEVM bool) (string, error) {
	w.Lock()
	defer w.Unlock()
	var err error

	txBuilder := w.EncCfg.TxConfig.NewTxBuilder()
	txBuilder.SetMemo("")

	if !isEVM {
		err = txBuilder.SetMsgs(msg)
		if err != nil {
			err = errors.Wrap(err, "SendTx, set msgs")
			return "", err
		}

		txBuilder.SetGasLimit(uint64(gasLimit))
		txBuilder.SetFeeAmount(sdktype.NewCoins(fee))
	} else {
		convertedMsg := msg.(*evmtypes.MsgEthereumTx)

		_, err = convertedMsg.BuildTx(txBuilder, xplatypes.DefaultDenom)
		if err != nil {
			err = errors.Wrap(err, "SendTx, build evm tx")
			return "", err
		}
	}

	sigV2 := signing.SignatureV2{
		PubKey: w.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode:  w.EncCfg.TxConfig.SignModeHandler().DefaultMode(),
			Signature: nil,
		},
		Sequence: w.Sequence,
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		err = errors.Wrap(err, "SendTx, SetSignatures")
		return "", err
	}

	signerData := xauthsigning.SignerData{
		ChainID:       chainId,
		AccountNumber: w.AccountNumber,
		Sequence:      w.Sequence,
	}

	sigV2, err = tx.SignWithPrivKey(
		w.EncCfg.TxConfig.SignModeHandler().DefaultMode(), signerData, txBuilder, w.PrivKey, w.EncCfg.TxConfig, w.Sequence)
	if err != nil {
		err = errors.Wrap(err, "SendTx, do sign")
		return "", err
	}

	err = txBuilder.SetSignatures(sigV2)
	if err != nil {
		err = errors.Wrap(err, "SendTx, set signatures")
		return "", err
	}

	txBytes, err := w.EncCfg.TxConfig.TxEncoder()(txBuilder.GetTx())
	if err != nil {
		err = errors.Wrap(err, "SendTx, tx byte encode")
		return "", err
	}

	txHash, err := BroadcastTx(desc.ServiceConn, w.ChainId, txBytes, txtype.BroadcastMode_BROADCAST_MODE_ASYNC)
	if err != nil {
		err = errors.Wrap(err, "SendTx, tx broadcast")
		return "", err
	}

	w.Sequence += 1

	return txHash, nil
}

func (w *WalletInfo) RefreshSequence() error {
	w.Lock()
	defer w.Unlock()

	accountNumber, seq, err := GetAccountNumber(desc.ServiceConn, w.ChainId, w.StringAddress)
	if err != nil {
		err = errors.Wrap(err, "RefreshSequence, get account info")
		return err
	}

	w.AccountNumber = accountNumber
	w.Sequence = seq

	return nil
}

func GRPCQueryContractStore(conn *grpc.ClientConn, chainId, contractAddress, param string) (*json.RawMessage, error) {
	client := cosmwasmtype.NewQueryClient(desc.ServiceConn)
	res, err := client.SmartContractState(context.Background(), &cosmwasmtype.QuerySmartContractStateRequest{
		Address:   contractAddress,
		QueryData: []byte(param),
	})

	if err != nil {
		err = errors.Wrap(err, "GRPCQueryContractStore")
		return nil, err
	}

	resData := json.RawMessage(res.Data.Bytes())

	return &resData, nil
}

func GenerateContractExecMessage(senderAddress, contractAddress string, param []byte, coins sdktype.Coins) *cosmwasmtype.MsgExecuteContract {
	return &cosmwasmtype.MsgExecuteContract{
		Sender:   senderAddress,
		Contract: contractAddress,
		Msg:      param,
		Funds:    coins,
	}

}

func GetAccountNumber(conn *grpc.ClientConn, chainId, address string) (uint64, uint64, error) {
	client := authtypes.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	res, err := client.Account(context.Background(), &authtypes.QueryAccountRequest{Address: address})
	if err != nil {
		err = errors.Wrap(err, "GetAccountNumber")
		return 0, 0, err
	}

	var baseAccount authtypes.ModuleAccount
	err = baseAccount.Unmarshal(res.Account.Value)
	if err != nil {
		err = errors.Wrap(err, "GetAccountNumber, unmarshalling")
		return 0, 0, err
	}

	return baseAccount.GetAccountNumber(), baseAccount.GetSequence(), nil
}

func BroadcastTx(conn *grpc.ClientConn, chainId string, txBytes []byte, mode txtype.BroadcastMode) (string, error) {
	queryTxClient := txtype.NewServiceClient(desc.GetConnectionWithContext(context.Background()))

	_, err := queryTxClient.Simulate(context.Background(), &txtype.SimulateRequest{
		TxBytes: txBytes,
	})

	if err != nil {
		return "", err
	}

	client := txtype.NewServiceClient(desc.ServiceConn)

	if currtestingenv := os.Getenv("GOLANG_TESTING"); currtestingenv != "true" {
		res, err := client.BroadcastTx(context.Background(), &txtype.BroadcastTxRequest{
			TxBytes: txBytes,
			Mode:    mode,
		})

		if err != nil {
			err = errors.Wrap(err, "broadcastTx")
			return "", err
		}

		return res.TxResponse.TxHash, nil
	}

	return "", nil
}
