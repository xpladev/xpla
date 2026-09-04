package wasm

import (
	"bytes"
	"context"
	"errors"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestContractMethodsRejectMissingAccount(t *testing.T) {
	contractAddress := common.HexToAddress("0x0000000000000000000000000000000000000001")
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	precompile := PrecompiledWasm{
		ak:  stubAccountKeeper{},
		wms: &stubWasmMsgServer{},
		wk:  &stubWasmKeeper{},
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
			sender, contractAddress, big.NewInt(1), []byte(`{}`),
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

func TestContractMethodsUseOriginalAccountAddress(t *testing.T) {
	originalAddress := sdk.AccAddress(bytes.Repeat([]byte{0x1}, 32))
	contractAddress := common.BytesToAddress(originalAddress)
	sender := common.HexToAddress("0x0000000000000000000000000000000000000002")
	accountKeeper := stubAccountKeeper{
		account: authtypes.NewBaseAccountWithAddress(originalAddress),
	}
	msgServerErr := errors.New("stop after capturing message")
	msgServer := &stubWasmMsgServer{err: msgServerErr}
	wasmKeeper := &stubWasmKeeper{err: errors.New("stop after capturing query")}
	precompile := PrecompiledWasm{
		ak:  accountKeeper,
		wms: msgServer,
		wk:  wasmKeeper,
	}

	_, err := precompile.executeContract(sdk.Context{}, nil, sender, nil, []interface{}{
		sender, contractAddress, []byte(`{}`), sdk.Coins{},
	})
	require.ErrorIs(t, err, msgServerErr)
	require.Equal(t, originalAddress.String(), msgServer.executeMsg.Contract)

	_, err = precompile.migrateContract(sdk.Context{}, nil, sender, nil, []interface{}{
		sender, contractAddress, big.NewInt(1), []byte(`{}`),
	})
	require.ErrorIs(t, err, msgServerErr)
	require.Equal(t, originalAddress.String(), msgServer.migrateMsg.Contract)

	_, err = precompile.smartContractState(sdk.Context{}, nil, []interface{}{
		contractAddress, []byte(`{}`),
	})
	require.ErrorIs(t, err, wasmKeeper.err)
	require.Equal(t, originalAddress, wasmKeeper.contractAddress)
}

type stubAccountKeeper struct {
	account sdk.AccountI
}

func (s stubAccountKeeper) GetAccount(context.Context, sdk.AccAddress) sdk.AccountI {
	return s.account
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
	err             error
	contractAddress sdk.AccAddress
}

func (s *stubWasmKeeper) QuerySmart(_ context.Context, contractAddress sdk.AccAddress, _ []byte) ([]byte, error) {
	s.contractAddress = contractAddress
	return nil, s.err
}
