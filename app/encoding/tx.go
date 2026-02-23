package encoding

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	txsigning "cosmossdk.io/x/tx/signing"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

var _ client.TxConfig = &TxConfigWrapper{}

// NewTxConfig creates a new TxConfigWrapper containing a custom TxDecoder that converts legacy messages
func NewTxConfig(cdc codec.Codec, sigModes []signingtypes.SignMode) *TxConfigWrapper {
	return &TxConfigWrapper{
		TxConfig: authtx.NewTxConfig(cdc, sigModes),
		Codec:    cdc,
	}
}

// TxConfigWrapper wraps the default TxConfig and provides a custom TxDecoder.
// TxConfig is a settable field so that the inner config can be replaced (e.g. with NewTxConfigWithOptions).
// Codec is used for the legacy MsgEthereumTx fallback decode path.
type TxConfigWrapper struct {
	TxConfig client.TxConfig
	Codec    codec.Codec
}

// TxDecoder returns a decoder that retries with LegacyTxDecodeRegistry when the first decode fails on legacy MsgEthereumTx (tag 1).
func (w *TxConfigWrapper) TxDecoder() sdk.TxDecoder {
	defaultDecoder := w.TxConfig.TxDecoder()

	return func(txBytes []byte) (sdk.Tx, error) {
		tx, err := defaultDecoder(txBytes)
		if err != nil && w.Codec != nil && IsLegacyMsgEthereumTxDecodeErr(err) {
			wrapped := NewEthereumTxCompatRegistry(w.Codec.InterfaceRegistry())
			fallbackDecoder := authtx.DefaultTxDecoder(codec.NewProtoCodec(LegacyTxDecodeRegistry(wrapped)))
			if fallbackTx, fallbackErr := fallbackDecoder(txBytes); fallbackErr == nil {
				tx, err = fallbackTx, nil
			}
		}
		if err != nil {
			return nil, err
		}
		return w.normalizeLegacyEthereumMsgs(tx)
	}
}

func (w *TxConfigWrapper) normalizeLegacyEthereumMsgs(tx sdk.Tx) (sdk.Tx, error) {
	msgs := tx.GetMsgs()
	newMsgs := make([]sdk.Msg, len(msgs))
	isModified := false

	for i, msg := range msgs {
		if legacyMsg, ok := msg.(*legacyevmtypes.MsgEthereumTx); ok {
			legacyTx := legacyMsg.AsTransaction()
			newMsg := &evmtypes.MsgEthereumTx{}
			newMsg.FromEthereumTx(legacyTx)
			if legacyTx != nil {
				signer := legacySigner(legacyTx.ChainId())
				_ = newMsg.FromSignedEthereumTx(legacyTx, signer)
			}
			newMsgs[i] = newMsg
			isModified = true
			continue
		}
		newMsgs[i] = msg
	}

	if !isModified {
		return tx, nil
	}

	txBuilder := w.NewTxBuilder()

	if feeTx, ok := tx.(sdk.FeeTx); ok {
		txBuilder.SetGasLimit(feeTx.GetGas())
		txBuilder.SetFeeAmount(feeTx.GetFee())
		txBuilder.SetFeePayer(feeTx.FeePayer())
		txBuilder.SetFeeGranter(feeTx.FeeGranter())
	}

	if memoTx, ok := tx.(sdk.TxWithMemo); ok {
		txBuilder.SetMemo(memoTx.GetMemo())
	}

	if tsTx, ok := tx.(sdk.TxWithTimeoutTimeStamp); ok {
		txBuilder.SetTimeoutTimestamp(tsTx.GetTimeoutTimeStamp())
	}

	if thTx, ok := tx.(sdk.TxWithTimeoutHeight); ok {
		txBuilder.SetTimeoutHeight(thTx.GetTimeoutHeight())
	}

	if uoTx, ok := tx.(sdk.TxWithUnordered); ok {
		txBuilder.SetUnordered(uoTx.GetUnordered())
	}

	if err := txBuilder.SetMsgs(newMsgs...); err != nil {
		return nil, err
	}

	if sigTx, ok := tx.(authsigning.SigVerifiableTx); ok {
		sigs, err := sigTx.GetSignaturesV2()
		if err != nil {
			return nil, err
		}
		if len(sigs) > 0 {
			if err := txBuilder.SetSignatures(sigs...); err != nil {
				return nil, err
			}
		}
	}

	if extTx, ok := tx.(authante.HasExtensionOptionsTx); ok {
		if extTxBuilder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder); ok {
			extTxBuilder.SetExtensionOptions(extTx.GetExtensionOptions()...)
			extTxBuilder.SetNonCriticalExtensionOptions(extTx.GetNonCriticalExtensionOptions()...)
		}
	}

	return txBuilder.GetTx(), nil
}

// TxEncoder delegates to the inner TxConfig
func (w *TxConfigWrapper) TxEncoder() sdk.TxEncoder {
	return w.TxConfig.TxEncoder()
}

// NewTxBuilder delegates to the inner TxConfig
func (w *TxConfigWrapper) NewTxBuilder() client.TxBuilder {
	return w.TxConfig.NewTxBuilder()
}

// SignModeHandler delegates to the inner TxConfig
func (w *TxConfigWrapper) SignModeHandler() *txsigning.HandlerMap {
	return w.TxConfig.SignModeHandler()
}

// TxJSONEncoder delegates to the inner TxConfig
func (w *TxConfigWrapper) TxJSONEncoder() sdk.TxEncoder {
	return w.TxConfig.TxJSONEncoder()
}

// TxJSONDecoder delegates to the inner TxConfig
func (w *TxConfigWrapper) TxJSONDecoder() sdk.TxDecoder {
	return w.TxConfig.TxJSONDecoder()
}

// MarshalSignatureJSON delegates to the inner TxConfig
func (w *TxConfigWrapper) MarshalSignatureJSON(sigs []signingtypes.SignatureV2) ([]byte, error) {
	return w.TxConfig.MarshalSignatureJSON(sigs)
}

// UnmarshalSignatureJSON delegates to the inner TxConfig
func (w *TxConfigWrapper) UnmarshalSignatureJSON(b []byte) ([]signingtypes.SignatureV2, error) {
	return w.TxConfig.UnmarshalSignatureJSON(b)
}

// WrapTxBuilder delegates to the inner TxConfig
func (w *TxConfigWrapper) WrapTxBuilder(tx sdk.Tx) (client.TxBuilder, error) {
	return w.TxConfig.WrapTxBuilder(tx)
}

// SigningContext delegates to the inner TxConfig
func (w *TxConfigWrapper) SigningContext() *txsigning.Context {
	return w.TxConfig.SigningContext()
}
