package encoding

import (
	"errors"
	"math/big"
	"strings"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/gogoproto/proto"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/cosmos/evm/x/vm/types"

	legacyevmtypes "github.com/xpladev/xpla/legacy/ethermint/x/evm/types"
)

const (
	msgEthereumTxTypeURL = "cosmos.evm.vm.v1.MsgEthereumTx"
	legacyTxTypeURL      = "cosmos.evm.vm.v1.LegacyTx"
	dynamicFeeTxTypeURL  = "cosmos.evm.vm.v1.DynamicFeeTx"
	accessListTxTypeURL  = "cosmos.evm.vm.v1.AccessListTx"
)

// EthereumTxCompatRegistry wraps an InterfaceRegistry and adds UnpackAny/Resolve
// behavior for pre-upgrade cosmos.evm.vm.v1.MsgEthereumTx (legacy payload).
type EthereumTxCompatRegistry struct {
	codectypes.InterfaceRegistry
}

// NewEthereumTxCompatRegistry returns a registry that delegates to inner except for
// UnpackAny and Resolve, where it applies Ethereum tx compatibility (legacy ↔ new).
// MsgEthereumTx is resolved via inner (new type) so response display shows from/raw.
func NewEthereumTxCompatRegistry(inner codectypes.InterfaceRegistry) *EthereumTxCompatRegistry {
	return &EthereumTxCompatRegistry{InterfaceRegistry: inner}
}

// Resolve implements AnyResolver (used by unknownproto.RejectUnknownFields and client tx decode).
//   - For cosmos.evm.vm.v1.MsgEthereumTx we delegate to inner (new type) so that response display
//     unmarshals correctly and shows from/raw. Legacy payloads (tag 1) fail first decode and are
//     retried in TxDecoder with LegacyTxDecodeRegistry.
//   - For cosmos.evm.vm.v1.LegacyTx/DynamicFeeTx/AccessListTx we return legacy TxData types so
//     the client can resolve inner Anys when decoding pre-upgrade txs (payload uses ethermint schema).
//   - All other type URLs are delegated to the inner registry as-is; the SDK stores type URLs
//     with a leading slash (e.g. /cosmos.gov.v1beta1.MsgSubmitProposal), so we must not strip it.
func (r *EthereumTxCompatRegistry) Resolve(typeURL string) (proto.Message, error) {
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

// UnpackAny implements AnyUnpacker. For cosmos.evm.vm.v1.MsgEthereumTx we try legacy
// unmarshal first; if it succeeds and has Data (pre-upgrade payload), convert to new with from/raw.
// For cosmos.evm.vm.v1.LegacyTx/DynamicFeeTx/AccessListTx unpacking into TxData we use legacy
// types so UnpackInterfaces on legacy MsgEthereumTx works (inner registry may not have
// cosmos.evm.vm.v1.* registered against ethermint TxData).
func (r *EthereumTxCompatRegistry) UnpackAny(any *codectypes.Any, iface interface{}) error {
	if any == nil {
		return r.InterfaceRegistry.UnpackAny(any, iface)
	}
	normalized := strings.TrimPrefix(any.TypeUrl, "/")

	// Unpack TxData (LegacyTx/DynamicFeeTx/AccessListTx) into legacyevmtypes.TxData so legacy MsgEthereumTx.UnpackInterfaces works.
	if ptr, ok := iface.(*legacyevmtypes.TxData); ok && ptr != nil {
		var concrete legacyevmtypes.TxData
		switch normalized {
		case legacyTxTypeURL:
			var legacy legacyevmtypes.LegacyTx
			if err := proto.Unmarshal(any.Value, &legacy); err != nil {
				return err
			}
			concrete = &legacy
		case dynamicFeeTxTypeURL:
			var legacy legacyevmtypes.DynamicFeeTx
			if err := proto.Unmarshal(any.Value, &legacy); err != nil {
				return err
			}
			concrete = &legacy
		case accessListTxTypeURL:
			var legacy legacyevmtypes.AccessListTx
			if err := proto.Unmarshal(any.Value, &legacy); err != nil {
				return err
			}
			concrete = &legacy
		default:
			concrete = nil
		}
		if concrete != nil {
			*ptr = concrete
			cached, err := codectypes.NewAnyWithValue(concrete.(proto.Message))
			if err != nil {
				return err
			}
			*any = *cached
			return nil
		}
	}

	if isCosmosEthereumTxTypeURL(any.TypeUrl) {
		var legacyMsg legacyevmtypes.MsgEthereumTx
		if errLegacy := proto.Unmarshal(any.Value, &legacyMsg); errLegacy == nil && legacyMsg.Data != nil {
			// Pre-upgrade payload (tag 1 = data): convert to new type with from/raw populated.
			if errUnpack := legacyMsg.UnpackInterfaces(r); errUnpack != nil {
				return errUnpack
			}
			legacyTx := legacyMsg.AsTransaction()
			newMsg := &evmtypes.MsgEthereumTx{}
			newMsg.FromEthereumTx(legacyTx)
			if legacyTx != nil {
				signer := legacySigner(legacyTx.ChainId())
				_ = newMsg.FromSignedEthereumTx(legacyTx, signer)
			}
			if ptr, ok := iface.(*sdk.Msg); ok {
				*ptr = newMsg
				cached, err := codectypes.NewAnyWithValue(newMsg)
				if err != nil {
					return err
				}
				*any = *cached
				return nil
			}
		}
	}
	return r.InterfaceRegistry.UnpackAny(any, iface)
}

// legacyTxDecodeRegistry overrides only Resolve for MsgEthereumTx (returns legacy type)
// so the fallback decode path passes RejectUnknownFields for tag 1 payloads.
type legacyTxDecodeRegistry struct {
	*EthereumTxCompatRegistry
}

// LegacyTxDecodeRegistry returns an InterfaceRegistry that resolves MsgEthereumTx as legacy type.
// Use only for the fallback decode when the first decode fails on legacy (tag 1) payload.
func LegacyTxDecodeRegistry(wrapped *EthereumTxCompatRegistry) codectypes.InterfaceRegistry {
	return &legacyTxDecodeRegistry{EthereumTxCompatRegistry: wrapped}
}

func (r *legacyTxDecodeRegistry) Resolve(typeURL string) (proto.Message, error) {
	normalized := strings.TrimPrefix(typeURL, "/")
	if normalized == msgEthereumTxTypeURL {
		return &legacyevmtypes.MsgEthereumTx{}, nil
	}
	return r.EthereumTxCompatRegistry.Resolve(typeURL)
}

func isCosmosEthereumTxTypeURL(typeURL string) bool {
	return strings.Contains(typeURL, msgEthereumTxTypeURL)
}

// legacySigner chooses a signer that won't panic for pre-EIP-155 signatures.
func legacySigner(chainID *big.Int) ethtypes.Signer {
	if chainID != nil && chainID.Sign() > 0 {
		return ethtypes.LatestSignerForChainID(chainID)
	}
	return ethtypes.HomesteadSigner{}
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
// Checks the full error chain in case the error is wrapped.
func IsLegacyMsgEthereumTxDecodeErr(err error) bool {
	for e := err; e != nil; e = errors.Unwrap(e) {
		s := e.Error()
		if strings.Contains(s, "MsgEthereumTx") &&
			strings.Contains(s, "TagNum: 1") &&
			isUnknownFieldProtoErr(e) {
			return true
		}
	}
	return false
}
