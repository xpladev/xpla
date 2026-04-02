package app

import (
	"math/big"
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	evmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/app/encoding"
	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

func TestNewXplaApp_ConsensusTxConfig_IsDefault(t *testing.T) {
	xpla := newTestApp(t)
	_, isCompat := xpla.GetTxConfig().(*encoding.TxConfigWrapper)
	require.False(t, isCompat)
}

func TestNewXplaApp_ConsensusTxDecoder_DoesNotNormalizeLegacyMsg(t *testing.T) {
	xpla := newTestApp(t)

	txBytes := legacyTxBytes(t, xpla.GetTxConfig(), big.NewInt(37))
	tx, err := xpla.GetTxConfig().TxDecoder()(txBytes)
	require.NoError(t, err)

	msgs := tx.GetMsgs()
	require.Len(t, msgs, 1)
	require.IsType(t, &legacyevmtypes.MsgEthereumTx{}, msgs[0])
	_, isNew := msgs[0].(*evmtypes.MsgEthereumTx)
	require.False(t, isNew)
}

func TestNewXplaApp_ConsensusTxDecoder_RejectsCosmosTypeURLLegacyPayload(t *testing.T) {
	xpla := newTestApp(t)

	txBytes := legacyTxBytes(t, xpla.GetTxConfig(), big.NewInt(37))
	legacyPayloadCosmosTypeURL := rewriteFirstTypeURL(t, txBytes, "/cosmos.evm.vm.v1.MsgEthereumTx")

	_, err := xpla.GetTxConfig().TxDecoder()(legacyPayloadCosmosTypeURL)
	require.Error(t, err)
	require.Contains(t, err.Error(), "TagNum: 1")
}

func newTestApp(t *testing.T) *XplaApp {
	t.Helper()
	home := t.TempDir()

	return NewXplaApp(
		log.NewNopLogger(),
		dbm.NewMemDB(),
		nil,
		true,
		map[int64]bool{},
		home,
		EmptyAppOptions{},
		EmptyWasmOptions,
	)
}

func legacyTxBytes(t *testing.T, txConfig client.TxConfig, v *big.Int) []byte {
	t.Helper()

	legacyMsg := newLegacyMsgEthereumTx(t, v)
	builder := txConfig.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(legacyMsg))

	txBytes, err := txConfig.TxEncoder()(builder.GetTx())
	require.NoError(t, err)
	return txBytes
}

func rewriteFirstTypeURL(t *testing.T, txBytes []byte, typeURL string) []byte {
	t.Helper()

	var raw txtypes.TxRaw
	require.NoError(t, proto.Unmarshal(txBytes, &raw))

	var body txtypes.TxBody
	require.NoError(t, proto.Unmarshal(raw.BodyBytes, &body))
	require.NotEmpty(t, body.Messages)

	body.Messages[0].TypeUrl = typeURL
	raw.BodyBytes = mustMarshalProto(t, &body)
	return mustMarshalProto(t, &raw)
}

func newLegacyMsgEthereumTx(t *testing.T, v *big.Int) *legacyevmtypes.MsgEthereumTx {
	t.Helper()

	ethTx := ethtypes.NewTx(&ethtypes.LegacyTx{
		Nonce:    7,
		GasPrice: big.NewInt(1),
		Gas:      21_000,
		Value:    big.NewInt(42),
		Data:     []byte{0x1, 0x2, 0x3},
		V:        new(big.Int).Set(v),
		R:        big.NewInt(1),
		S:        big.NewInt(1),
	})

	txData, err := legacyevmtypes.NewTxDataFromTx(ethTx)
	require.NoError(t, err)

	txDataProto, ok := txData.(proto.Message)
	require.True(t, ok)

	anyData, err := codectypes.NewAnyWithValue(txDataProto)
	require.NoError(t, err)
	return &legacyevmtypes.MsgEthereumTx{Data: anyData}
}

func mustMarshalProto(t *testing.T, msg proto.Message) []byte {
	t.Helper()

	bz, err := proto.Marshal(msg)
	require.NoError(t, err)
	return bz
}
