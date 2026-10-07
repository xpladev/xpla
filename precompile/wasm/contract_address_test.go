package wasm

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestContractMethodsRejectMissingAccount(t *testing.T) {
	contractAddress := common.HexToAddress("0x0000000000000000000000000000000000000001")
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	precompile := PrecompiledWasm{
		wms: &stubWasmMsgServer{},
		wk:  &stubWasmKeeper{resolveErr: wasmtypes.ErrNoSuchContractFn(sdk.AccAddress(contractAddress.Bytes()).String())},
	}
	expectedErr := wasmtypes.ErrNoSuchContractFn(sdk.AccAddress(contractAddress.Bytes()).String())

	t.Run("execute", func(t *testing.T) {
		_, err := precompile.executeContract(sdk.Context{}, nil, sender, nil, []interface{}{
			sender, contractAddress, []byte(`{}`), sdk.Coins{},
		})

		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("migrate", func(t *testing.T) {
		_, err := precompile.migrateContract(sdk.Context{}, nil, sender, nil, []interface{}{
			sender, contractAddress, uint64(1), []byte(`{}`),
		})

		require.EqualError(t, err, expectedErr.Error())
	})

	t.Run("smart query", func(t *testing.T) {
		_, err := precompile.smartContractState(sdk.Context{}, nil, []interface{}{
			contractAddress, []byte(`{}`),
		})

		require.EqualError(t, err, expectedErr.Error())
	})
}

func TestContractMethodsUseRegisteredWasmAddress(t *testing.T) {
	originalAddress := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 32))
	contractAddress := common.BytesToAddress(originalAddress)
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	msgServerErr := errors.New("stop after capturing message")
	msgServer := &stubWasmMsgServer{err: msgServerErr}
	wasmKeeper := &stubWasmKeeper{
		resolvedAddress: originalAddress,
		err:             errors.New("stop after capturing query"),
	}
	precompile := PrecompiledWasm{
		wms: msgServer,
		wk:  wasmKeeper,
	}

	_, err := precompile.executeContract(sdk.Context{}, nil, sender, nil, []interface{}{
		sender, contractAddress, []byte(`{}`), sdk.Coins{},
	})
	require.ErrorIs(t, err, msgServerErr)
	require.Equal(t, originalAddress.String(), msgServer.executeMsg.Contract)

	_, err = precompile.migrateContract(sdk.Context{}, nil, sender, nil, []interface{}{
		sender, contractAddress, uint64(1), []byte(`{}`),
	})
	require.ErrorIs(t, err, msgServerErr)
	require.Equal(t, originalAddress.String(), msgServer.migrateMsg.Contract)

	_, err = precompile.smartContractState(sdk.Context{}, nil, []interface{}{
		contractAddress, []byte(`{}`),
	})
	require.ErrorIs(t, err, wasmKeeper.err)
	require.Equal(t, originalAddress, wasmKeeper.contractAddress)
}

func TestContractMethodsPreserveExactTwentyByteWasmAddress(t *testing.T) {
	contractAddress := common.HexToAddress("0x0000000000000000000000000000000000000001")
	exactAddress := sdk.AccAddress(contractAddress.Bytes())
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	msgServerErr := errors.New("stop after capturing message")
	msgServer := &stubWasmMsgServer{err: msgServerErr}
	precompile := PrecompiledWasm{
		wms: msgServer,
		wk:  &stubWasmKeeper{resolvedAddress: exactAddress},
	}

	_, err := precompile.executeContract(sdk.Context{}, nil, sender, nil, []interface{}{
		sender, contractAddress, []byte(`{}`), sdk.Coins{},
	})
	require.ErrorIs(t, err, msgServerErr)
	require.Equal(t, exactAddress.String(), msgServer.executeMsg.Contract)
}

type stubWasmMsgServer struct {
	err        error
	executeMsg *wasmtypes.MsgExecuteContract
	migrateMsg *wasmtypes.MsgMigrateContract
}

func (*stubWasmMsgServer) InstantiateContract(context.Context, *wasmtypes.MsgInstantiateContract) (*wasmtypes.MsgInstantiateContractResponse, error) {
	panic("unexpected call")
}

func (*stubWasmMsgServer) InstantiateContract2(context.Context, *wasmtypes.MsgInstantiateContract2) (*wasmtypes.MsgInstantiateContract2Response, error) {
	panic("unexpected call")
}

func (s *stubWasmMsgServer) ExecuteContract(_ context.Context, msg *wasmtypes.MsgExecuteContract) (*wasmtypes.MsgExecuteContractResponse, error) {
	s.executeMsg = msg
	return nil, s.err
}

func (s *stubWasmMsgServer) MigrateContract(_ context.Context, msg *wasmtypes.MsgMigrateContract) (*wasmtypes.MsgMigrateContractResponse, error) {
	s.migrateMsg = msg
	return nil, s.err
}

type stubWasmKeeper struct {
	resolvedAddress sdk.AccAddress
	resolveErr      error
	err             error
	contractAddress sdk.AccAddress
}

func (s *stubWasmKeeper) ResolveContractAddress(context.Context, sdk.AccAddress) (sdk.AccAddress, error) {
	return s.resolvedAddress, s.resolveErr
}

func (s *stubWasmKeeper) QuerySmart(_ context.Context, contractAddress sdk.AccAddress, _ []byte) ([]byte, error) {
	s.contractAddress = contractAddress
	return nil, s.err
}
