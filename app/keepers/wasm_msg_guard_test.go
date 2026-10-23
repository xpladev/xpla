package keepers

import (
	"bytes"
	"testing"

	storetypes "cosmossdk.io/store/types"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	vmtypes "github.com/cosmos/evm/x/vm/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"
)

func TestValidateNoEVMMsg(t *testing.T) {
	addr := sdk.AccAddress(bytes.Repeat([]byte{0x11}, 20))
	evm := &vmtypes.MsgEthereumTx{}

	t.Run("direct current EVM message", func(t *testing.T) {
		err := validateNoEVMMsg(evm, maxWasmAnyScanDepth)
		requireDisabledEVMError(t, err)
	})

	t.Run("authz MsgExec", func(t *testing.T) {
		exec := authz.NewMsgExec(addr, []sdk.Msg{evm})
		err := validateNoEVMMsg(&exec, maxWasmAnyScanDepth)
		requireDisabledEVMError(t, err)
	})

	t.Run("authz MsgGrant", func(t *testing.T) {
		grant, err := authz.NewMsgGrant(
			addr,
			addr,
			authz.NewGenericAuthorization(evmMsgTypeURL),
			nil,
		)
		require.NoError(t, err)
		requireDisabledEVMError(t, validateNoEVMMsg(grant, maxWasmAnyScanDepth))
	})

	t.Run("gov proposal", func(t *testing.T) {
		proposal, err := govv1.NewMsgSubmitProposal(
			[]sdk.Msg{evm}, nil, addr.String(), "", "blocked", "blocked", false,
		)
		require.NoError(t, err)
		requireDisabledEVMError(t, validateNoEVMMsg(proposal, maxWasmAnyScanDepth))
	})

	t.Run("gov proposal nested in authz", func(t *testing.T) {
		proposal, err := govv1.NewMsgSubmitProposal(
			[]sdk.Msg{evm}, nil, addr.String(), "", "blocked", "blocked", false,
		)
		require.NoError(t, err)
		exec := authz.NewMsgExec(addr, []sdk.Msg{proposal})
		requireDisabledEVMError(t, validateNoEVMMsg(&exec, maxWasmAnyScanDepth))
	})

	t.Run("authz nested in gov proposal", func(t *testing.T) {
		exec := authz.NewMsgExec(addr, []sdk.Msg{evm})
		proposal, err := govv1.NewMsgSubmitProposal(
			[]sdk.Msg{&exec}, nil, addr.String(), "", "blocked", "blocked", false,
		)
		require.NoError(t, err)
		requireDisabledEVMError(t, validateNoEVMMsg(proposal, maxWasmAnyScanDepth))
	})

	t.Run("non EVM message", func(t *testing.T) {
		err := validateNoEVMMsg(&banktypes.MsgSend{}, maxWasmAnyScanDepth)
		require.NoError(t, err)
	})

	t.Run("recursion depth", func(t *testing.T) {
		var nested sdk.Msg = evm
		for range maxWasmAnyScanDepth + 1 {
			exec := authz.NewMsgExec(addr, []sdk.Msg{nested})
			nested = &exec
		}
		err := validateNoEVMMsg(nested, maxWasmAnyScanDepth)
		require.ErrorContains(t, err, "recursion depth")
	})
}

func TestWasmAnyEVMEncoder(t *testing.T) {
	c := newGuardTestCodec()
	addr := sdk.AccAddress(bytes.Repeat([]byte{0x22}, 20))
	evm := &vmtypes.MsgEthereumTx{}
	encoder := newWasmAnyEVMEncoder(c)
	encode := func(msg wasmvmtypes.AnyMsg) ([]sdk.Msg, error) {
		ctx := sdk.Context{}.WithGasMeter(storetypes.NewInfiniteGasMeter())
		return encoder(ctx, addr, &msg)
	}

	t.Run("current EVM URL with and without leading slash", func(t *testing.T) {
		for _, typeURL := range []string{evmMsgTypeURL, "/" + evmMsgTypeURL} {
			t.Run(typeURL, func(t *testing.T) {
				msg := marshalWasmAny(t, typeURL, evm)
				_, err := encode(msg)
				requireDisabledEVMError(t, err)
			})
		}
	})

	t.Run("malformed disabled EVM payload is rejected by URL", func(t *testing.T) {
		_, err := encode(wasmvmtypes.AnyMsg{
			TypeURL: "/" + evmMsgTypeURL,
			Value:   []byte{0xff},
		})
		requireDisabledEVMError(t, err)
	})

	t.Run("authz MsgExec", func(t *testing.T) {
		exec := authz.NewMsgExec(addr, []sdk.Msg{evm})
		_, err := encode(marshalWasmAny(t, sdk.MsgTypeURL(&exec), &exec))
		requireDisabledEVMError(t, err)
	})

	t.Run("authz MsgGrant", func(t *testing.T) {
		grant, err := authz.NewMsgGrant(
			addr,
			addr,
			authz.NewGenericAuthorization(evmMsgTypeURL),
			nil,
		)
		require.NoError(t, err)
		_, err = encode(marshalWasmAny(t, sdk.MsgTypeURL(grant), grant))
		requireDisabledEVMError(t, err)
	})

	t.Run("gov proposal containing authz", func(t *testing.T) {
		exec := authz.NewMsgExec(addr, []sdk.Msg{evm})
		proposal, err := govv1.NewMsgSubmitProposal(
			[]sdk.Msg{&exec}, nil, addr.String(), "", "blocked", "blocked", false,
		)
		require.NoError(t, err)
		_, err = encode(marshalWasmAny(t, sdk.MsgTypeURL(proposal), proposal))
		requireDisabledEVMError(t, err)
	})

	t.Run("authz containing unregistered message fails closed", func(t *testing.T) {
		unknownAny := &codectypes.Any{TypeUrl: "/xpla.test.Unknown"}
		exec := &authz.MsgExec{Grantee: addr.String(), Msgs: []*codectypes.Any{unknownAny}}
		_, err := encode(marshalWasmAny(t, sdk.MsgTypeURL(exec), exec))
		require.Error(t, err)
	})

	t.Run("decode and unpack failures fail closed", func(t *testing.T) {
		cases := map[string]wasmvmtypes.AnyMsg{
			"empty type URL": {},
			"malformed known type": {
				TypeURL: sdk.MsgTypeURL(&banktypes.MsgSend{}),
				Value:   []byte{0xff},
			},
			"unknown type": {
				TypeURL: "/xpla.test.Unknown",
				Value:   nil,
			},
		}
		for name, msg := range cases {
			t.Run(name, func(t *testing.T) {
				_, err := encode(msg)
				require.Error(t, err)
			})
		}
	})

	t.Run("non EVM Any is accepted", func(t *testing.T) {
		msg := marshalWasmAny(t, sdk.MsgTypeURL(&banktypes.MsgSend{}), &banktypes.MsgSend{})
		msgs, err := encode(msg)
		require.NoError(t, err)
		require.Len(t, msgs, 1)
		require.IsType(t, &banktypes.MsgSend{}, msgs[0])
	})

	t.Run("non EVM Any uses the standard gas charge", func(t *testing.T) {
		msg := marshalWasmAny(t, sdk.MsgTypeURL(&banktypes.MsgSend{}), &banktypes.MsgSend{})
		ctx := sdk.Context{}.WithGasMeter(storetypes.NewInfiniteGasMeter())
		_, err := encoder(ctx, addr, &msg)
		require.NoError(t, err)
		require.Equal(
			t,
			wasmkeeper.AnyMsgBaseGasCost+
				wasmkeeper.AnyMsgPerKBGasCost*uint64(len(msg.Value))/1024,
			ctx.GasMeter().GasConsumed(),
		)
	})

	t.Run("oversized non EVM Any uses the standard size limit", func(t *testing.T) {
		_, err := encode(wasmvmtypes.AnyMsg{
			TypeURL: sdk.MsgTypeURL(&banktypes.MsgSend{}),
			Value:   make([]byte, wasmkeeper.MaxAnyMsgValueSize+1),
		})
		require.ErrorContains(t, err, "exceeds limit")
	})
}

func newGuardTestCodec() codec.Codec {
	registry := codectypes.NewInterfaceRegistry()
	authz.RegisterInterfaces(registry)
	banktypes.RegisterInterfaces(registry)
	govv1.RegisterInterfaces(registry)
	vmtypes.RegisterInterfaces(registry)
	return codec.NewProtoCodec(registry)
}

func marshalWasmAny(t *testing.T, typeURL string, msg proto.Message) wasmvmtypes.AnyMsg {
	t.Helper()
	bz, err := proto.Marshal(msg)
	require.NoError(t, err)
	return wasmvmtypes.AnyMsg{TypeURL: typeURL, Value: bz}
}

func requireDisabledEVMError(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	require.Contains(t, err.Error(), "found disabled msg type")
	require.Contains(t, err.Error(), "/"+evmMsgTypeURL)
}
