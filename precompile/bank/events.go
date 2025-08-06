package bank

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	EventTypeSend = "Send"
)

// EmitSendEvent creates a new send event emitted on a send transaction.
func (p PrecompiledBank) EmitSendEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	from common.Address,
	to common.Address,
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

	// convert sdk.Coin to util.Coin and generate the data field and pack
	abiCoins := cmn.NewCoinsResponse(amount)
	packedData, err := event.Inputs.NonIndexed().Pack(abiCoins)
	if err != nil {
		return fmt.Errorf("EmitSendEvent: failed to pack event data: %w", err)
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}
