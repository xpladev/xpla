package encoding

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	signingtypes "github.com/cosmos/cosmos-sdk/types/tx/signing"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

var _ client.TxConfig = &TxConfigWrapper{}

// NewTxConfig creates a new TxConfigWrapper containing a custom TxDecoder that converts legacy messages
func NewTxConfig(cdc codec.Codec, sigModes []signingtypes.SignMode) *TxConfigWrapper {
	return &TxConfigWrapper{
		TxConfig: authtx.NewTxConfig(cdc, sigModes),
	}
}

// TxConfigWrapper wraps the default TxConfig and provides a custom TxDecoder
type TxConfigWrapper struct {
	client.TxConfig
}

// The TxDecoder wraps the default TxDecoder to convert the legacy ethermint MsgEthereumTx to the new cosmos evmtypes MsgEthereumTx
func (w *TxConfigWrapper) TxDecoder() sdk.TxDecoder {
	defaultDecoder := w.TxConfig.TxDecoder()

	return func(txBytes []byte) (sdk.Tx, error) {
		tx, err := defaultDecoder(txBytes)
		if err != nil {
			return nil, err
		}

		msgs := tx.GetMsgs()
		newMsgs := make([]sdk.Msg, len(msgs))
		isModified := false

		for i, msg := range msgs {
			// convert legacy ethermint message into the evmtypes message
			if legacyMsg, ok := msg.(*legacyevmtypes.MsgEthereumTx); ok {
				legacyTx := legacyMsg.AsTransaction()
				newMsg := &evmtypes.MsgEthereumTx{}
				if err := newMsg.FromEthereumTx(legacyTx); err != nil {
					return nil, fmt.Errorf("failed to convert legacy msg at index %d: %w", i, err)
				}
				newMsgs[i] = newMsg
				isModified = true
			} else {
				newMsgs[i] = msg
			}
		}

		// if a conversion occurred, the transaction is rebuilt
		if isModified {
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

			if sigTx, ok := tx.(authsigning.SigVerifiableTx); ok {
				sigs, err := sigTx.GetSignaturesV2()
				if err != nil {
					return nil, err
				}
				txBuilder.SetSignatures(sigs...)
			}

			if extTx, ok := tx.(authante.HasExtensionOptionsTx); ok {
				if extTxBuilder, ok := txBuilder.(authtx.ExtensionOptionsTxBuilder); ok {
					extTxBuilder.SetExtensionOptions(extTx.GetExtensionOptions()...)
					extTxBuilder.SetNonCriticalExtensionOptions(extTx.GetNonCriticalExtensionOptions()...)
				}
			}

			if err := txBuilder.SetMsgs(newMsgs...); err != nil {
				return nil, err
			}

			return txBuilder.GetTx(), nil
		}

		return tx, nil
	}
}
