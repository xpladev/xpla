package encoding

import (
	txsigning "cosmossdk.io/x/tx/signing"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
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
		if err != nil && w.Codec != nil {
			if wrapped, ok := w.Codec.InterfaceRegistry().(*EthereumTxCompatRegistry); ok && IsLegacyMsgEthereumTxDecodeErr(err) {
				fallbackDecoder := authtx.DefaultTxDecoder(codec.NewProtoCodec(LegacyTxDecodeRegistry(wrapped)))
				if fallbackTx, fallbackErr := fallbackDecoder(txBytes); fallbackErr == nil {
					tx, err = fallbackTx, nil
				}
			}
		}
		if err != nil {
			return nil, err
		}
		return tx, nil
	}
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
