package wasm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"
	cmn "github.com/cosmos/evm/precompiles/common"
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
	sender, admin, contractAddress common.Address,
	codeId *big.Int,
	label string,
) (err error) {
	event := p.Events[EventTypeInstantiateContract]

	// prepare event topics
	topics := make([]common.Hash, 4)
	topics[0] = event.ID
	topics[1], err = cmn.MakeTopic(sender)
	if err != nil {
		return err
	}
	topics[2], err = cmn.MakeTopic(admin)
	if err != nil {
		return err
	}
	topics[3], err = cmn.MakeTopic(contractAddress)
	if err != nil {
		return err
	}

	// pack data fields
	packedData, err := event.Inputs.NonIndexed().Pack(codeId, label)
	if err != nil {
		return fmt.Errorf("failed to pack event data: %w", err)
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
	sender, contractAddress common.Address,
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

	// For the sake of brevity, detailed information such as `msg` and `funds` is omitted
	// so empty non-indexed parameter
	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        []byte{},
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}

// EmitMigrateContractEvent creates a new event emitted on MigrateContract
func (p PrecompiledWasm) EmitMigrateContractEvent(
	ctx sdk.Context,
	stateDB vm.StateDB,
	sender, contractAddress common.Address,
	codeId *big.Int,
) (err error) {
	event := p.Events[EventTypeMigrateContract]

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

	// pack data fields
	packedData, err := event.Inputs.NonIndexed().Pack(codeId)
	if err != nil {
		return fmt.Errorf("failed to pack event data: %w", err)
	}

	stateDB.AddLog(&ethtypes.Log{
		Address:     p.Address(),
		Topics:      topics,
		Data:        packedData,
		BlockNumber: uint64(ctx.BlockHeight()),
	})

	return nil
}
