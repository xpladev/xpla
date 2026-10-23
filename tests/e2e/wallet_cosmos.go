package e2e

import (
	"context"
	"sync"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
)

type WalletInfo struct {
	sync.Mutex

	IsSrc         bool
	ChainId       string
	Prefix        string
	StringAddress string
	ByteAddress   sdk.AccAddress
	PrivKey       cryptotypes.PrivKey
	PubKey        cryptotypes.PubKey
	AccountNumber uint64
	Sequence      uint64
	EncCfg        moduletestutil.TestEncodingConfig
}

// SignTx signs and encodes a transaction using the wallet's account number and
// sequence. It leaves gas, fees, broadcasting and sequence updates to the caller.
func (w *WalletInfo) SignTx(ctx context.Context, chainID string, builder client.TxBuilder) ([]byte, error) {
	if err := builder.SetSignatures(signing.SignatureV2{
		PubKey: w.PrivKey.PubKey(),
		Data: &signing.SingleSignatureData{
			SignMode: signing.SignMode_SIGN_MODE_DIRECT,
		},
		Sequence: w.Sequence,
	}); err != nil {
		return nil, errors.Wrap(err, "SignTx, set signer")
	}

	signature, err := tx.SignWithPrivKey(ctx, signing.SignMode_SIGN_MODE_DIRECT, authsigning.SignerData{
		ChainID:       chainID,
		AccountNumber: w.AccountNumber,
		Sequence:      w.Sequence,
	}, builder, w.PrivKey, w.EncCfg.TxConfig, w.Sequence)
	if err != nil {
		return nil, errors.Wrap(err, "SignTx, sign")
	}
	if err := builder.SetSignatures(signature); err != nil {
		return nil, errors.Wrap(err, "SignTx, set signature")
	}
	return w.EncCfg.TxConfig.TxEncoder()(builder.GetTx())
}

func GetAccountNumber(ctx context.Context, conn grpc.ClientConnInterface, address string) (uint64, uint64, error) {
	res, err := authtypes.NewQueryClient(conn).AccountInfo(ctx, &authtypes.QueryAccountInfoRequest{Address: address})
	if err != nil {
		return 0, 0, errors.Wrap(err, "GetAccountNumber")
	}
	return res.Info.GetAccountNumber(), res.Info.GetSequence(), nil
}

// BroadcastTx returns the full response, including nonzero CheckTx codes, so
// callers can assert either successful submission or the expected rejection.
func BroadcastTx(ctx context.Context, conn grpc.ClientConnInterface, txBytes []byte, mode txtypes.BroadcastMode) (*sdk.TxResponse, error) {
	res, err := txtypes.NewServiceClient(conn).BroadcastTx(ctx, &txtypes.BroadcastTxRequest{
		TxBytes: txBytes,
		Mode:    mode,
	})
	if err != nil {
		return nil, errors.Wrap(err, "broadcastTx")
	}
	return res.TxResponse, nil
}
