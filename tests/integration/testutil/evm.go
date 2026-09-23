package testutil

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"math/big"
	"testing"
	"time"

	storetypes "cosmossdk.io/store/types"
	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/evm/x/vm/statedb"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"

	xplatypes "github.com/xpladev/xpla/types"
)

// DefaultEVMGasLimit is the gas budget for integration contract calls.
const DefaultEVMGasLimit = uint64(10_000_000)

// ContractArtifact contains a compiled contract's ABI and deployment bytecode.
type ContractArtifact struct {
	ABI      json.RawMessage `json:"abi"`
	Bytecode string          `json:"bytecode"`
}

// EVMTransactionResult contains the response, receipt and SDK events of a committed transaction.
type EVMTransactionResult struct {
	Response vmtypes.MsgEthereumTxResponse
	Receipt  *ethtypes.Receipt
	Events   []abci.Event
}

// EVMReceiptRecorder collects receipts when explicitly installed as an EVM keeper hook.
type EVMReceiptRecorder struct {
	receipts []*ethtypes.Receipt
}

// PostTxProcessing records the outer transaction receipt and copies its log slice.
func (r *EVMReceiptRecorder) PostTxProcessing(
	_ sdk.Context,
	_ common.Address,
	_ core.Message,
	receipt *ethtypes.Receipt,
) error {
	copyReceipt := *receipt
	copyReceipt.Logs = append([]*ethtypes.Log(nil), receipt.Logs...)
	r.receipts = append(r.receipts, &copyReceipt)
	return nil
}

func (r *EVMReceiptRecorder) get(hash common.Hash) *ethtypes.Receipt {
	for _, receipt := range r.receipts {
		if receipt.TxHash == hash {
			return receipt
		}
	}
	return nil
}

// EVMLogsForEvent selects logs emitted by an address with the given event signature.
func EVMLogsForEvent(receipt *ethtypes.Receipt, address common.Address, topic common.Hash) []*ethtypes.Log {
	var logs []*ethtypes.Log
	for _, log := range receipt.Logs {
		if log.Address == address && len(log.Topics) > 0 && log.Topics[0] == topic {
			logs = append(logs, log)
		}
	}
	return logs
}

// EVMContext reads the latest committed state with a fresh gas meter.
// Its synthetic timestamp matches CommitTransaction's block-height clock.
func (ti *TestInput) EVMContext() sdk.Context {
	height := ti.App.LastBlockHeight()
	return ti.App.BaseApp.NewUncachedContext(false, tmproto.Header{
		ChainID: ti.Ctx.ChainID(), Height: height, Time: time.Unix(height, 0).UTC(),
		ProposerAddress: ti.Ctx.BlockHeader().ProposerAddress,
	}).WithGasMeter(storetypes.NewGasMeter(DefaultEVMGasLimit))
}

// CommitEVMTransaction signs and commits a transaction through the real block lifecycle.
// EVM execution failures are returned for rollback assertions; SDK failures fail the test.
func (ti *TestInput) CommitEVMTransaction(t *testing.T, key *ecdsa.PrivateKey, receipts *EVMReceiptRecorder, to *common.Address, data []byte, gas uint64) EVMTransactionResult {
	t.Helper()
	sender := crypto.PubkeyToAddress(key.PublicKey)
	nonce := ti.App.EvmKeeper.GetNonce(ti.EVMContext(), sender)
	signer := ethtypes.LatestSignerForChainID(vmtypes.GetEthChainConfig().ChainID)
	tx, err := ethtypes.SignNewTx(key, signer, &ethtypes.LegacyTx{
		Nonce: nonce, GasPrice: big.NewInt(1_000_000_000), Gas: gas, To: to, Value: big.NewInt(0), Data: data,
	})
	require.NoError(t, err)
	var msg vmtypes.MsgEthereumTx
	require.NoError(t, msg.FromSignedEthereumTx(tx, signer))
	cosmosTx, err := msg.BuildTx(ti.App.GetTxConfig().NewTxBuilder(), xplatypes.DefaultDenom)
	require.NoError(t, err)
	txBytes, err := ti.App.GetTxConfig().TxEncoder()(cosmosTx)
	require.NoError(t, err)
	result := ti.CommitTransaction(t, txBytes)
	receipt := receipts.get(tx.Hash())
	require.NotNil(t, receipt, "committed transaction must have an outer receipt")
	return EVMTransactionResult{Response: decodeEVMResponse(t, result), Receipt: receipt, Events: result.Events}
}

// DeployEVMContract commits a contract deployment and requires successful EVM execution.
func (ti *TestInput) DeployEVMContract(t *testing.T, key *ecdsa.PrivateKey, receipts *EVMReceiptRecorder, artifacts map[string]ContractArtifact, name string, args ...interface{}) (common.Address, abi.ABI) {
	t.Helper()
	artifact, ok := artifacts[name]
	require.True(t, ok, "missing fixture contract %s", name)
	contractABI, err := abi.JSON(bytes.NewReader(artifact.ABI))
	require.NoError(t, err)
	constructor, err := contractABI.Pack("", args...)
	require.NoError(t, err)
	nonce := ti.App.EvmKeeper.GetNonce(ti.EVMContext(), crypto.PubkeyToAddress(key.PublicKey))
	address := crypto.CreateAddress(crypto.PubkeyToAddress(key.PublicKey), nonce)
	result := ti.CommitEVMTransaction(t, key, receipts, nil, append(common.FromHex(artifact.Bytecode), constructor...), DefaultEVMGasLimit)
	require.Empty(t, result.Response.VmError)
	require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
	return address, contractABI
}

// CommitEVMContractCall commits an ABI-encoded call and requires successful EVM execution.
func (ti *TestInput) CommitEVMContractCall(t *testing.T, key *ecdsa.PrivateKey, receipts *EVMReceiptRecorder, address common.Address, contractABI abi.ABI, method string, args ...interface{}) EVMTransactionResult {
	t.Helper()
	data, err := contractABI.Pack(method, args...)
	require.NoError(t, err)
	result := ti.CommitEVMTransaction(t, key, receipts, &address, data, DefaultEVMGasLimit)
	require.Empty(t, result.Response.VmError)
	require.Equal(t, uint64(ethtypes.ReceiptStatusSuccessful), result.Receipt.Status)
	return result
}

// QueryEVMContract executes and decodes a contract call without committing state changes.
func (ti *TestInput) QueryEVMContract(t *testing.T, from common.Address, address common.Address, contractABI abi.ABI, method string, args ...interface{}) []interface{} {
	t.Helper()
	ctx := ti.EVMContext()
	result, err := ti.App.EvmKeeper.CallEVM(ctx, statedb.New(ctx, ti.App.EvmKeeper, statedb.NewEmptyTxConfig()),
		contractABI, from, address, false, false, big.NewInt(int64(DefaultEVMGasLimit)), method, args...)
	require.NoError(t, err)
	require.Empty(t, result.VmError)
	values, err := contractABI.Unpack(method, result.Ret)
	require.NoError(t, err)
	return values
}

func decodeEVMResponse(t *testing.T, result *abci.ExecTxResult) vmtypes.MsgEthereumTxResponse {
	t.Helper()
	var txMsgData sdk.TxMsgData
	require.NoError(t, proto.Unmarshal(result.Data, &txMsgData))
	require.Len(t, txMsgData.MsgResponses, 1)
	var response vmtypes.MsgEthereumTxResponse
	require.NoError(t, proto.Unmarshal(txMsgData.MsgResponses[0].Value, &response))
	return response
}
