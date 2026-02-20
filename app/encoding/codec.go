package encoding

import (
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

const (
	msgEthereumTxTypeURL   = "cosmos.evm.vm.v1.MsgEthereumTx"
	legacyTxTypeURL        = "cosmos.evm.vm.v1.LegacyTx"
	dynamicFeeTxTypeURL    = "cosmos.evm.vm.v1.DynamicFeeTx"
	accessListTxTypeURL    = "cosmos.evm.vm.v1.AccessListTx"
)

// RegistryWithEthereumTxFallback wraps an InterfaceRegistry and adds UnpackAny
// fallback for pre-upgrade cosmos.evm.vm.v1.MsgEthereumTx (legacy payload).
// Use this so that any code path that unpacks via the registry (e.g. x/tx decode)
// gets the fallback.
type RegistryWithEthereumTxFallback struct {
	codectypes.InterfaceRegistry
}

// NewRegistryWithEthereumTxFallback returns a registry that delegates to inner
// except for UnpackAny and Resolve, where it applies the EthereumTx fallback.
func NewRegistryWithEthereumTxFallback(inner codectypes.InterfaceRegistry) *RegistryWithEthereumTxFallback {
	return &RegistryWithEthereumTxFallback{InterfaceRegistry: inner}
}

// Resolve implements AnyResolver (used by unknownproto.RejectUnknownFields and client tx decode).
// - For cosmos.evm.vm.v1.MsgEthereumTx we delegate to the inner registry so the new (post-upgrade)
//   type is used; RejectUnknownFields then validates against the new schema (tag 5=from, 6=raw).
// - For cosmos.evm.vm.v1.LegacyTx/DynamicFeeTx/AccessListTx we return legacy TxData types so
//   the client can resolve inner Anys when decoding pre-upgrade txs (payload uses ethermint schema).
// - All other type URLs are delegated to the inner registry as-is; the SDK stores type URLs
//   with a leading slash (e.g. /cosmos.gov.v1beta1.MsgSubmitProposal), so we must not strip it.
func (r *RegistryWithEthereumTxFallback) Resolve(typeURL string) (proto.Message, error) {
	normalized := strings.TrimPrefix(typeURL, "/")
	switch normalized {
	case legacyTxTypeURL:
		return &legacyevmtypes.LegacyTx{}, nil
	case dynamicFeeTxTypeURL:
		return &legacyevmtypes.DynamicFeeTx{}, nil
	case accessListTxTypeURL:
		return &legacyevmtypes.AccessListTx{}, nil
	}
	return r.InterfaceRegistry.Resolve(typeURL)
}

// UnpackAny implements AnyUnpacker. For cosmos.evm.vm.v1.MsgEthereumTx with
// legacy payload, retries unmarshal as legacy MsgEthereumTx and converts to new.
func (r *RegistryWithEthereumTxFallback) UnpackAny(any *codectypes.Any, iface interface{}) error {
	err := r.InterfaceRegistry.UnpackAny(any, iface)
	if err == nil {
		return nil
	}
	if any == nil || !isCosmosEthereumTxTypeURL(any.TypeUrl) || !isUnknownFieldProtoErr(err) {
		return err
	}
	var legacyMsg legacyevmtypes.MsgEthereumTx
	if errLegacy := proto.Unmarshal(any.Value, &legacyMsg); errLegacy != nil {
		return err
	}
	if legacyMsg.Data != nil {
		var txData legacyevmtypes.TxData
		if errInner := r.InterfaceRegistry.UnpackAny(legacyMsg.Data, &txData); errInner != nil {
			return err
		}
	}
	newMsg := &evmtypes.MsgEthereumTx{}
	newMsg.FromEthereumTx(legacyMsg.AsTransaction())
	if ptr, ok := iface.(*sdk.Msg); ok {
		*ptr = newMsg
		return nil
	}
	return err
}

func isCosmosEthereumTxTypeURL(typeURL string) bool {
	return strings.Contains(typeURL, msgEthereumTxTypeURL)
}

func isUnknownFieldProtoErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "unknown field") ||
		strings.Contains(s, "errUnknownField") ||
		strings.Contains(s, "illegal tag") ||
		strings.Contains(s, "TagNum") ||
		strings.Contains(s, "parse error")
}

// IsLegacyMsgEthereumTxDecodeErr returns true when the error is from RejectUnknownFields
// failing on a legacy MsgEthereumTx payload (tag 1 = data). Used to trigger fallback decode.
func IsLegacyMsgEthereumTxDecodeErr(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, "MsgEthereumTx") &&
		strings.Contains(s, "TagNum: 1") &&
		isUnknownFieldProtoErr(err)
}

// LegacyFirstRegistry wraps a RegistryWithEthereumTxFallback and overrides Resolve so that
// cosmos.evm.vm.v1.MsgEthereumTx returns the legacy type. Used only for the fallback decode
// path so RejectUnknownFields accepts legacy (tag 1) payloads; UnpackAny still converts to new.
type LegacyFirstRegistry struct {
	*RegistryWithEthereumTxFallback
}

// NewLegacyFirstRegistry returns a registry that behaves like wrapped except Resolve
// returns the legacy MsgEthereumTx type for cosmos.evm.vm.v1.MsgEthereumTx.
func NewLegacyFirstRegistry(wrapped *RegistryWithEthereumTxFallback) *LegacyFirstRegistry {
	return &LegacyFirstRegistry{RegistryWithEthereumTxFallback: wrapped}
}

// Resolve returns legacy MsgEthereumTx for that type URL so RejectUnknownFields passes for legacy payloads.
func (r *LegacyFirstRegistry) Resolve(typeURL string) (proto.Message, error) {
	normalized := strings.TrimPrefix(typeURL, "/")
	if normalized == msgEthereumTxTypeURL {
		return &legacyevmtypes.MsgEthereumTx{}, nil
	}
	return r.RegistryWithEthereumTxFallback.Resolve(typeURL)
}
