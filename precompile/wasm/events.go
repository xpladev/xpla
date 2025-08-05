package wasm

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

const (
	EventTypeInstantiateContract = "InstantiateContract"
	EventTypeExecuteContract     = "ExecuteContract"
	EventTypeMigrateContract     = "MigrateContract"
)

// EmitInstantiateContractEvent creates a new event emitted on InstantiateContract, InstantiateContract2
func (p PrecompiledWasm) EmitInstantiateContractEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender common.Address,
	admin common.Address,
	contractAddress common.Address,
	codeId *big.Int,
	label string,
	msg []byte,
	funds sdk.Coins,
	data []byte,
) (err error) {
	event := p.Events[EventTypeInstantiateContract]

	// prepare event topics
	topics := make([]common.Hash, 4)
	topics[0] = event.ID
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(contractAddress)
	if err != nil {
		return err
	}
	topics[3], err = cmn.MakeTopic(codeId)
	if err != nil {
		return err
	}

	// convert sdk.Coin to util.Coin and generate the data field and pack
	abiCoins := cmn.NewCoinsResponse(funds)
	packedData, err := event.Inputs.NonIndexed().Pack(admin, label, msg, abiCoins, data)
	if err != nil {
		return fmt.Errorf("EmitInstantiateContractEvent: failed to pack event data: %w", err)
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}

// EmitExecuteContractEvent creates a new event emitted on ExecuteContract
func (p PrecompiledWasm) EmitExecuteContractEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender common.Address,
	contractAddress common.Address,
	msg []byte,
	funds sdk.Coins,
	data []byte,
) (err error) {
	event := p.Events[EventTypeExecuteContract]

	// prepare event topics
	topics := make([]common.Hash, 3)
	topics[0] = event.ID
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(contractAddress)
	if err != nil {
		return err
	}

	// convert sdk.Coin to util.Coin and generate the data field and pack
	abiCoins := cmn.NewCoinsResponse(funds)
	packedData, err := event.Inputs.NonIndexed().Pack(msg, abiCoins, data)
	if err != nil {
		return fmt.Errorf("EmitExecuteContractEvent: failed to pack event data: %w", err)
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}

// EmitMigrateContractEvent creates a new event emitted on MigrateContract
func (p PrecompiledWasm) EmitMigrateContractEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender common.Address,
	contractAddress common.Address,
	codeId *big.Int,
	msg []byte,
	data []byte,
) (err error) {
	event := p.Events[EventTypeMigrateContract]

	// prepare event topics
	topics := make([]common.Hash, 4)
	topics[0] = event.ID
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(contractAddress)
	if err != nil {
		return err
	}
	topics[3], err = cmn.MakeTopic(codeId)
	if err != nil {
		return err
	}

	// pack data fields
	packedData, err := event.Inputs.NonIndexed().Pack(msg, data)
	if err != nil {
		return fmt.Errorf("EmitMigrateContractEvent: failed to pack event data: %w", err)
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}
