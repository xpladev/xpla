package bank

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	cmn "github.com/cosmos/evm/precompiles/common"
)

// Coin for converting sdk.Coin Amount from sdkmath.Int to *big.Int
type Coin struct {
	Denom  string
	Amount *big.Int
}

const (
	EventTypeSend = "Send"
)

// EmitSendEvent creates a new send event emitted on a send transaction.
func (p PrecompiledBank) EmitSendEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	from, to common.Address,
	amount sdk.Coins,
) (err error) {
	event := p.Events[EventTypeSend]

	// prepare event topics
	topics := make([]common.Hash, 3)
	topics[0] = event.ID
	topics[1], err = cmn.MakeTopic(from)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(to)
	if err != nil {
		return err
	}

	// generate the data field and pack
	abiCoins := make([]Coin, len(amount))
	for i, coin := range amount {
		abiCoins[i] = Coin{
			Denom:  coin.Denom,
			Amount: coin.Amount.BigInt(),
		}
	}
	packedData, err := event.Inputs.NonIndexed().Pack(abiCoins)
	if err != nil {
		return err
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}
