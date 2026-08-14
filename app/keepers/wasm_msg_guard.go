package keepers

import (
	"fmt"
	"strings"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
)

const (
	evmMsgTypeURL       = "cosmos.evm.vm.v1.MsgEthereumTx"
	maxWasmAnyScanDepth = 8
)

// validateNoEVMMsg rejects EVM messages nested in SDK container types after
// the standard Wasm Any encoder has unpacked their interfaces.
func validateNoEVMMsg(msg sdk.Msg, depth int) error {
	if msg == nil {
		return fmt.Errorf("cannot scan nil sdk message")
	}

	if err := evmMsgTypeError(sdk.MsgTypeURL(msg)); err != nil {
		return err
	}

	var (
		nested        []sdk.Msg
		containerName string
		err           error
	)
	switch container := msg.(type) {
	case *authz.MsgExec:
		if len(container.Msgs) == 0 {
			return nil
		}
		if depth <= 0 {
			return fmt.Errorf("wasm Any message recursion depth exceeded")
		}
		nested, err = container.GetMessages()
		containerName = "authz MsgExec"
	case *authz.MsgGrant:
		authorization, err := container.GetAuthorization()
		if err != nil {
			return fmt.Errorf("unpack authz MsgGrant authorization: %w", err)
		}
		return evmMsgTypeError(authorization.MsgTypeURL())
	case *govv1.MsgSubmitProposal:
		if len(container.Messages) == 0 {
			return nil
		}
		if depth <= 0 {
			return fmt.Errorf("wasm Any message recursion depth exceeded")
		}
		nested, err = container.GetMsgs()
		containerName = "gov MsgSubmitProposal"
	default:
		return nil
	}
	if err != nil {
		return fmt.Errorf("unpack %s messages: %w", containerName, err)
	}
	for _, msg := range nested {
		if err := validateNoEVMMsg(msg, depth-1); err != nil {
			return err
		}
	}

	return nil
}

func newWasmAnyEVMEncoder(appCodec codec.Codec) wasmkeeper.AnyEncoder {
	defaultEncoder := wasmkeeper.EncodeAnyMsg(appCodec)
	return func(
		ctx sdk.Context,
		sender sdk.AccAddress,
		msg *wasmvmtypes.AnyMsg,
	) ([]sdk.Msg, error) {
		if err := evmMsgTypeError(msg.TypeURL); err != nil {
			return nil, err
		}

		msgs, err := defaultEncoder(ctx, sender, msg)
		if err != nil {
			return nil, err
		}
		for _, sdkMsg := range msgs {
			if err := validateNoEVMMsg(sdkMsg, maxWasmAnyScanDepth); err != nil {
				return nil, err
			}
		}
		return msgs, nil
	}
}

func evmMsgTypeError(typeURL string) error {
	typeURL = strings.TrimPrefix(typeURL, "/")
	if typeURL != evmMsgTypeURL {
		return nil
	}
	return fmt.Errorf("found disabled msg type /%s", typeURL)
}
