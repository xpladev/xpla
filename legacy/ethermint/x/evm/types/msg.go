// Copyright 2021 Evmos Foundation
// This file is part of Evmos' Ethermint library.
//
// The Ethermint library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The Ethermint library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the Ethermint library. If not, see https://github.com/xpladev/ethermint/blob/main/LICENSE
package types

import (
	"errors"
	"fmt"
	"math/big"

	protov2 "google.golang.org/protobuf/proto"

	errorsmod "cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/xpladev/xpla/legacy/ethermint/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	interfaces "github.com/cosmos/evm/rpc/types/interfaces"
)

var (
	_ sdk.Msg                   = &MsgEthereumTx{}
	_ sdk.Tx                    = &MsgEthereumTx{}
	_ ante.GasTx                = &MsgEthereumTx{}
	_ interfaces.IMsgEthereumTx = &MsgEthereumTx{}

	_ codectypes.UnpackInterfacesMessage = MsgEthereumTx{}
)

// message type and route constants
const (
	// TypeMsgEthereumTx defines the type string of an Ethereum transaction
	TypeMsgEthereumTx = "ethereum_tx"
)

// FromEthereumTx populates the message fields from the given ethereum transaction
func (msg *MsgEthereumTx) FromEthereumTx(tx *ethtypes.Transaction) {
	txData, err := NewTxDataFromTx(tx)
	if err != nil {
		panic(err)
	}
	anyTxData, err := PackTxData(txData)
	if err != nil {
		panic(err)
	}

	msg.Data = anyTxData
	msg.Hash = tx.Hash().Hex()
}

// FromSignedEthereumTx populates the message fields from the given signed ethereum transaction, and set From field.
func (msg *MsgEthereumTx) FromSignedEthereumTx(tx *ethtypes.Transaction, signer ethtypes.Signer) error {
	from, err := ethtypes.Sender(signer, tx)
	if err != nil {
		return err
	}

	msg.From = from.Hex()
	return nil
}

// Route returns the route value of an MsgEthereumTx.
func (msg MsgEthereumTx) Route() string { return "" }

// Type returns the type value of an MsgEthereumTx.
func (msg MsgEthereumTx) Type() string { return TypeMsgEthereumTx }

// ErrGasOverflow returns an error if gas computation overlow/underflow
// ValidateBasic implements the sdk.Msg interface. It performs basic validation
// checks of a Transaction. If returns an error if validation fails.
func (msg MsgEthereumTx) ValidateBasic() error {
	if msg.From != "" {
		if err := types.ValidateAddress(msg.From); err != nil {
			return errorsmod.Wrap(err, "invalid from address")
		}
	}

	// Validate Size_ field, should be kept empty
	if msg.Size_ != 0 {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "tx size is deprecated")
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return errorsmod.Wrap(err, "failed to unpack tx data")
	}

	gas := txData.GetGas()

	// prevent txs with 0 gas to fill up the mempool
	if gas == 0 {
		return errorsmod.Wrap(errors.New("ErrInvalidGasLimit"), "gas limit must not be zero")
	}

	// prevent gas limit from overflow
	if g := new(big.Int).SetUint64(gas); !g.IsInt64() {
		return errorsmod.Wrap(errors.New("ErrGasOverflow"), "gas limit must be less than math.MaxInt64")
	}

	if err := txData.Validate(); err != nil {
		return err
	}

	// Validate Hash field after validated txData to avoid panic
	txHash := msg.AsTransaction().Hash().Hex()
	if msg.Hash != txHash {
		return errorsmod.Wrapf(errortypes.ErrInvalidRequest, "invalid tx hash %s, expected: %s", msg.Hash, txHash)
	}

	return nil
}

// GetMsgs returns a single MsgEthereumTx as an sdk.Msg.
func (msg *MsgEthereumTx) GetMsgs() []sdk.Msg {
	return []sdk.Msg{msg}
}

// GetMsgsV2 gets the transaction's messages as google.golang.org/protobuf/proto.Message's.
func (msg *MsgEthereumTx) GetMsgsV2() ([]protov2.Message, error) { return nil, nil }

// GetSigners returns the expected signers for an Ethereum transaction message.
// For such a message, there should exist only a single 'signer'.
//
// NOTE: This method panics if 'Sign' hasn't been called first.
func (msg *MsgEthereumTx) GetSigners() []sdk.AccAddress {
	data, err := UnpackTxData(msg.Data)
	if err != nil {
		panic(err)
	}

	sender, err := msg.GetSender(data.GetChainID())
	if err != nil {
		panic(err)
	}

	signer := sdk.AccAddress(sender.Bytes())
	return []sdk.AccAddress{signer}
}

// GetSignBytes returns the Amino bytes of an Ethereum transaction message used
// for signing.
//
// NOTE: This method cannot be used as a chain ID is needed to create valid bytes
// to sign over. Use 'RLPSignBytes' instead.
func (msg MsgEthereumTx) GetSignBytes() []byte {
	panic("must use 'RLPSignBytes' with a chain ID to get the valid bytes to sign")
}

// Sign calculates a secp256k1 ECDSA signature and signs the transaction. It
// takes a keyring signer and the chainID to sign an Ethereum transaction according to
// EIP155 standard.
// This method mutates the transaction as it populates the V, R, S
// fields of the Transaction's Signature.
// The function will fail if the sender address is not defined for the msg or if
// the sender is not registered on the keyring
func (msg *MsgEthereumTx) Sign(ethSigner ethtypes.Signer, keyringSigner keyring.Signer) error {
	from := msg.GetFrom()
	if from.Empty() {
		return fmt.Errorf("sender address not defined for message")
	}

	tx := msg.AsTransaction()
	txHash := ethSigner.Hash(tx)

	sig, _, err := keyringSigner.SignByAddress(from, txHash.Bytes(), signing.SignMode_SIGN_MODE_TEXTUAL)
	if err != nil {
		return err
	}

	tx, err = tx.WithSignature(ethSigner, sig)
	if err != nil {
		return err
	}

	return msg.FromSignedEthereumTx(tx, ethSigner)
}

// GetGas implements the GasTx interface. It returns the GasLimit of the transaction.
func (msg MsgEthereumTx) GetGas() uint64 {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return 0
	}
	return txData.GetGas()
}

// GetFee returns the fee for non dynamic fee tx
func (msg MsgEthereumTx) GetFee() *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.Fee()
}

// GetEffectiveFee returns the fee for dynamic fee tx
func (msg MsgEthereumTx) GetEffectiveFee(baseFee *big.Int) *big.Int {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}
	return txData.EffectiveFee(baseFee)
}

// GetFrom loads the ethereum sender address from the sigcache and returns an
// sdk.AccAddress from its bytes
func (msg *MsgEthereumTx) GetFrom() sdk.AccAddress {
	if msg.From == "" {
		return nil
	}

	return common.HexToAddress(msg.From).Bytes()
}

// AsTransaction creates an Ethereum Transaction type from the msg fields
func (msg MsgEthereumTx) AsTransaction() *ethtypes.Transaction {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}

	return ethtypes.NewTx(txData.AsEthereumData())
}

// AsMessage creates an Ethereum core.Message from the msg fields
func (msg MsgEthereumTx) AsMessage(baseFee *big.Int) *core.Message {
	tx := msg.AsTransaction()
	ethMsg := &core.Message{
		Nonce:                 tx.Nonce(),
		GasLimit:              tx.Gas(),
		GasPrice:              new(big.Int).Set(tx.GasPrice()),
		GasFeeCap:             new(big.Int).Set(tx.GasFeeCap()),
		GasTipCap:             new(big.Int).Set(tx.GasTipCap()),
		To:                    tx.To(),
		Value:                 tx.Value(),
		Data:                  tx.Data(),
		AccessList:            tx.AccessList(),
		SetCodeAuthorizations: tx.SetCodeAuthorizations(),
		SkipNonceChecks:       false,
		SkipFromEOACheck:      false,
		BlobHashes:            tx.BlobHashes(),
		BlobGasFeeCap:         tx.BlobGasFeeCap(),
	}
	// If baseFee provided, set gasPrice to effectiveGasPrice.
	if baseFee != nil {
		ethMsg.GasPrice = ethMsg.GasPrice.Add(ethMsg.GasTipCap, baseFee)
		if ethMsg.GasPrice.Cmp(ethMsg.GasFeeCap) > 0 {
			ethMsg.GasPrice = ethMsg.GasFeeCap
		}
	}

	var err error
	ethMsg.From, err = msg.GetSender(tx.ChainId())
	if err != nil {
		return nil
	}
	return ethMsg
}

// GetSender extracts the sender address from the signature values using the latest signer for the given chainID.
func (msg *MsgEthereumTx) GetSender(chainID *big.Int) (common.Address, error) {
	signer := ethtypes.LatestSignerForChainID(chainID)
	from, err := signer.Sender(msg.AsTransaction())
	if err != nil {
		return common.Address{}, err
	}

	msg.From = from.Hex()
	return from, nil
}

// GetSenderLegacy fallbacks to old behavior if From is empty, should be used by json-rpc
func (msg *MsgEthereumTx) GetSenderLegacy(signer ethtypes.Signer) (common.Address, error) {
	sender, err := msg.recoverSender(signer)
	if err != nil {
		return common.Address{}, err
	}
	msg.From = sender.Hex()
	return sender, nil
}

// UnpackInterfaces implements UnpackInterfacesMesssage.UnpackInterfaces
func (msg MsgEthereumTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	return unpacker.UnpackAny(msg.Data, new(TxData))
}

// recoverSender recovers the sender address from the transaction signature.
func (msg *MsgEthereumTx) recoverSender(signer ethtypes.Signer) (common.Address, error) {
	return ethtypes.Sender(signer, msg.AsTransaction())
}

// UnmarshalBinary decodes the canonical encoding of transactions.
func (msg *MsgEthereumTx) UnmarshalBinary(b []byte, signer ethtypes.Signer) error {
	tx := &ethtypes.Transaction{}
	if err := tx.UnmarshalBinary(b); err != nil {
		return err
	}
	return msg.FromSignedEthereumTx(tx, signer)
}

// BuildTx builds the canonical cosmos tx from ethereum msg
func (msg *MsgEthereumTx) BuildTx(b client.TxBuilder, evmDenom string) (authsigning.Tx, error) {
	builder, ok := b.(authtx.ExtensionOptionsTxBuilder)
	if !ok {
		return nil, errors.New("unsupported builder")
	}

	option, err := codectypes.NewAnyWithValue(&ExtensionOptionsEthereumTx{})
	if err != nil {
		return nil, err
	}

	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil, err
	}
	fees := make(sdk.Coins, 0)
	feeAmt := sdkmath.NewIntFromBigInt(txData.Fee())
	if feeAmt.Sign() > 0 {
		fees = append(fees, sdk.NewCoin(evmDenom, feeAmt))
	}

	builder.SetExtensionOptions(option)

	// A valid msg should have empty `From`
	msg.From = ""

	err = builder.SetMsgs(msg)
	if err != nil {
		return nil, err
	}
	builder.SetFeeAmount(fees)
	builder.SetGasLimit(msg.GetGas())
	tx := builder.GetTx()
	return tx, nil
}
