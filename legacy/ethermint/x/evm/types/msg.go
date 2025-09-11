package types

import (
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
)

var _ codectypes.UnpackInterfacesMessage = (*MsgEthereumTx)(nil)

func (msg *MsgEthereumTx) UnpackInterfaces(unpacker codectypes.AnyUnpacker) error {
	if msg.Data == nil {
		return nil
	}

	var data TxData
	return unpacker.UnpackAny(msg.Data, &data)
}

// AsTransaction creates an Ethereum Transaction type from the msg fields
func (msg MsgEthereumTx) AsTransaction() *ethtypes.Transaction {
	txData, err := UnpackTxData(msg.Data)
	if err != nil {
		return nil
	}

	return ethtypes.NewTx(txData.AsEthereumData())
}
