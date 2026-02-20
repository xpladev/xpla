package encoding

import (
	"math/big"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	txpb "github.com/cosmos/cosmos-sdk/types/tx"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/cosmos/gogoproto/proto"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

func TestTxDecoder_ConvertsEthermintTypeURLToNewMsg(t *testing.T) {
	txConfig := newTestTxConfig()

	txBytes := makeSingleMsgTxBytes(t, makeLegacyEthereumMsgAny(t, false, big.NewInt(37)))
	tx, err := txConfig.TxDecoder()(txBytes)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		t.Fatalf("unexpected msg count: %d", len(msgs))
	}

	if _, ok := msgs[0].(*evmtypes.MsgEthereumTx); !ok {
		t.Fatalf("expected new MsgEthereumTx, got %T", msgs[0])
	}
	if _, ok := msgs[0].(*legacyevmtypes.MsgEthereumTx); ok {
		t.Fatalf("expected conversion from legacy type URL, got legacy %T", msgs[0])
	}
}

func TestTxDecoder_UnprotectedLegacySignatureDoesNotPanic(t *testing.T) {
	txConfig := newTestTxConfig()

	// v=27 is a pre-EIP-155 signature (no chain-id); this used to panic in signer selection.
	txBytes := makeSingleMsgTxBytes(t, makeLegacyEthereumMsgAny(t, true, big.NewInt(27)))

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("unexpected panic for unprotected legacy signature: %v", r)
		}
	}()

	tx, err := txConfig.TxDecoder()(txBytes)
	if err != nil {
		t.Fatalf("decode failed: %v", err)
	}

	msgs := tx.GetMsgs()
	if len(msgs) != 1 {
		t.Fatalf("unexpected msg count: %d", len(msgs))
	}
	if _, ok := msgs[0].(*evmtypes.MsgEthereumTx); !ok {
		t.Fatalf("expected new MsgEthereumTx, got %T", msgs[0])
	}
}

func newTestTxConfig() *TxConfigWrapper {
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	interfaceRegistry = NewEthereumTxCompatRegistry(interfaceRegistry)

	evmtypes.RegisterInterfaces(interfaceRegistry)
	legacyevmtypes.RegisterInterfaces(interfaceRegistry)

	cdc := codec.NewProtoCodec(interfaceRegistry)
	return NewTxConfig(cdc, authtx.DefaultSignModes)
}

func makeLegacyEthereumMsgAny(t *testing.T, cosmosTypeURL bool, v *big.Int) *codectypes.Any {
	t.Helper()

	gasPrice := sdkmath.NewInt(1)
	amount := sdkmath.NewInt(0)
	legacyTxData := &legacyevmtypes.LegacyTx{
		Nonce:    1,
		GasPrice: &gasPrice,
		GasLimit: 21000,
		To:       "0x0000000000000000000000000000000000000000",
		Amount:   &amount,
		Data:     []byte{},
		V:        v.Bytes(),
		R:        big.NewInt(1).Bytes(),
		S:        big.NewInt(1).Bytes(),
	}
	legacyTxDataAny, err := codectypes.NewAnyWithValue(legacyTxData)
	if err != nil {
		t.Fatalf("pack legacy txdata: %v", err)
	}

	legacyMsg := &legacyevmtypes.MsgEthereumTx{Data: legacyTxDataAny}
	legacyMsgAny, err := codectypes.NewAnyWithValue(legacyMsg)
	if err != nil {
		t.Fatalf("pack legacy msg: %v", err)
	}

	if cosmosTypeURL {
		// Keep legacy payload but route through current type URL.
		legacyMsgAny.TypeUrl = "/" + msgEthereumTxTypeURL
	}
	return legacyMsgAny
}

func makeSingleMsgTxBytes(t *testing.T, msgAny *codectypes.Any) []byte {
	t.Helper()

	body := txpb.TxBody{Messages: []*codectypes.Any{msgAny}}
	bodyBz, err := proto.Marshal(&body)
	if err != nil {
		t.Fatalf("marshal body: %v", err)
	}

	authInfo := txpb.AuthInfo{}
	authBz, err := proto.Marshal(&authInfo)
	if err != nil {
		t.Fatalf("marshal auth info: %v", err)
	}

	raw := txpb.TxRaw{
		BodyBytes:     bodyBz,
		AuthInfoBytes: authBz,
	}
	rawBz, err := proto.Marshal(&raw)
	if err != nil {
		t.Fatalf("marshal tx raw: %v", err)
	}
	return rawBz
}
