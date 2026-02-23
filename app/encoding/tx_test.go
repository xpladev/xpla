package encoding

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"
	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

// 1) Legacy — ethermint.evm.v1.MsgEthereumTx (data, hash, from)
func makeLegacyEthremintMsgEthereumTx() *codectypes.Any {
	gasPrice := sdkmath.NewInt(1000000000)
	amount := sdkmath.ZeroInt()
	legacyTx := &legacyevmtypes.LegacyTx{
		Nonce:    0,
		GasPrice: &gasPrice,
		GasLimit: 21000,
		To:       "",
		Amount:   &amount,
		Data:     []byte{},
	}
	dataAny, err := codectypes.NewAnyWithValue(legacyTx)
	if err != nil {
		panic(err)
	}

	msg := &legacyevmtypes.MsgEthereumTx{
		Data: dataAny,
		Hash: "0x010203",
		From: "0x0000000000000000000000000000000000000001",
	}
	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}
	return msgAny
}

// 2) Legacy — ethermint.evm.v1.MsgEthereumTx (data, hash, from)
func makeLegacyCosmosEvmMsgEthereumTx() *codectypes.Any {
	gasPrice := sdkmath.NewInt(1000000000)
	amount := sdkmath.ZeroInt()
	legacyTx := &legacyevmtypes.LegacyTx{
		Nonce:    0,
		GasPrice: &gasPrice,
		GasLimit: 21000,
		To:       "",
		Amount:   &amount,
		Data:     []byte{0x01, 0x02, 0x03, 0x04},
	}

	dataAny, err := codectypes.NewAnyWithValue(legacyTx)
	if err != nil {
		panic(err)
	}
	dataAny.TypeUrl = "/cosmos.evm.vm.v1.LegacyTx"

	msg := &legacyevmtypes.MsgEthereumTx{
		Data: dataAny,
		Hash: "0x010203",
		From: "0x0000000000000000000000000000000000000001",
	}
	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}
	msgAny.TypeUrl = "/cosmos.evm.vm.v1.MsgEthereumTx"
	return msgAny
}

// 3) New — cosmos.evm.vm.v1.MsgEthereumTx (from, raw)
func makeNewMsgEthereumTx() *codectypes.Any {
	msg := &evmtypes.MsgEthereumTx{
		From: []byte{0x01, 0x02, 0x03, 0x04},
	}
	msgAny, err := codectypes.NewAnyWithValue(msg)
	if err != nil {
		panic(err)
	}
	return msgAny
}

var (
	anyLegacyMsgEthereumTx            *codectypes.Any // 1: TypeUrl=ethermint, Value=legacy bytes
	anyCosmosTypeURLWithLegacyPayload *codectypes.Any // 2: TypeUrl=cosmos, Value=legacy bytes
	anyNewMsgEthereumTx               *codectypes.Any // 3: TypeUrl=cosmos, Value=new bytes
)

func init() {
	anyLegacyMsgEthereumTx = makeLegacyEthremintMsgEthereumTx()
	anyCosmosTypeURLWithLegacyPayload = makeLegacyCosmosEvmMsgEthereumTx()
	anyNewMsgEthereumTx = makeNewMsgEthereumTx()
}

func makeTestCodec() codec.Codec {
	inner := codectypes.NewInterfaceRegistry()
	evmtypes.RegisterInterfaces(inner)
	legacyevmtypes.RegisterInterfaces(inner)
	wrapped := NewEthereumTxCompatRegistry(inner)
	return codec.NewProtoCodec(wrapped)
}

func TestUnpackAny_ThreeMsgTypes(t *testing.T) {
	cdc := makeTestCodec()

	t.Run("legacy_MsgEthereumTx_unpacks_as_legacy", func(t *testing.T) {
		var msg sdk.Msg
		err := cdc.UnpackAny(anyLegacyMsgEthereumTx, &msg)
		require.NoError(t, err)
		require.NotNil(t, msg)
		legacyMsg, ok := msg.(*legacyevmtypes.MsgEthereumTx)
		require.True(t, ok, "expected *legacyevmtypes.MsgEthereumTx, got %T", msg)
		require.NotNil(t, legacyMsg.Data, "legacy msg should have data")
		require.Equal(t, "0x010203", legacyMsg.Hash)
		require.Equal(t, "0x0000000000000000000000000000000000000001", legacyMsg.From)

		var inner proto.Message
		err = cdc.UnpackAny(legacyMsg.Data, &inner)
		require.NoError(t, err)
		legacyTx, ok := inner.(*legacyevmtypes.LegacyTx)
		require.True(t, ok, "expected *legacyevmtypes.LegacyTx, got %T", inner)
		require.Equal(t, uint64(0), legacyTx.Nonce)
		require.True(t, legacyTx.GasPrice.Equal(sdkmath.NewInt(1000000000)))
		require.Equal(t, uint64(21000), legacyTx.GasLimit)
	})

	t.Run("cosmos_typeURL_with_legacy_payload_unpacks_as_legacy", func(t *testing.T) {
		var msg sdk.Msg
		err := cdc.UnpackAny(anyCosmosTypeURLWithLegacyPayload, &msg)
		require.NoError(t, err)
		require.NotNil(t, msg)
		unpacked, ok := msg.(*legacyevmtypes.MsgEthereumTx)
		require.True(t, ok, "cosmos typeURL + legacy payload는 *legacyevmtypes.MsgEthereumTx가 나와야 함, got %T", msg)
		require.NotNil(t, unpacked.Data)
		require.Equal(t, "0x010203", unpacked.Hash)
		require.Equal(t, "0x0000000000000000000000000000000000000001", unpacked.From)
		var inner proto.Message
		err = cdc.UnpackAny(unpacked.Data, &inner)
		require.NoError(t, err)
		legacyTx, ok := inner.(*legacyevmtypes.LegacyTx)
		require.True(t, ok, "expected *legacyevmtypes.LegacyTx, got %T", inner)
		require.Equal(t, uint64(0), legacyTx.Nonce)
		require.True(t, legacyTx.GasPrice.Equal(sdkmath.NewInt(1000000000)))
		require.Equal(t, uint64(21000), legacyTx.GasLimit)
	})

	t.Run("new_MsgEthereumTx_unpacks_as_new", func(t *testing.T) {
		var msg sdk.Msg
		err := cdc.UnpackAny(anyNewMsgEthereumTx, &msg)
		require.NoError(t, err)
		require.NotNil(t, msg)
		unpacked, ok := msg.(*evmtypes.MsgEthereumTx)
		require.True(t, ok, "expected *evmtypes.MsgEthereumTx, got %T", msg)
		require.Equal(t, []byte{0x01, 0x02, 0x03, 0x04}, unpacked.From)
	})
}
