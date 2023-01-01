package keeper

import (
	context "context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/armon/go-metrics"
	"github.com/cosmos/cosmos-sdk/telemetry"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	tmbytes "github.com/tendermint/tendermint/libs/bytes"
	tmtypes "github.com/tendermint/tendermint/types"
	"github.com/xpladev/xpla/x/proxyevm/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the reward MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

func (k msgServer) CallEVM(goCtx context.Context, msg *types.MsgCallEVM) (*evmtypes.MsgEthereumTxResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	txIndex := k.evmKeeper.GetTxIndexTransient(ctx)

	fmt.Println("keeper, GetTxIndexTransient", ctx.GasMeter().GasConsumed())

	var labels []metrics.Label
	sender, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}

	var contract *common.Address
	if msg.Contract != "" {
		addr := common.HexToAddress(msg.Contract)
		contract = &addr

		labels = []metrics.Label{
			telemetry.NewLabel("execution", "call"),
			telemetry.NewLabel("to", contract.Hex()), // recipient address (contract or account)
		}
	} else {
		labels = []metrics.Label{telemetry.NewLabel("execution", "create")}
	}

	fmt.Println("keeper, Address processing", ctx.GasMeter().GasConsumed())

	evmParams := k.evmKeeper.GetParams(ctx)

	fmt.Println("keeper, GetParams", ctx.GasMeter().GasConsumed())

	fundAmount := sdk.ZeroInt()
	for _, coin := range msg.Funds {
		if coin.Denom == evmParams.EvmDenom {
			fundAmount = coin.Amount
		}
	}

	fmt.Println("keeper, msg.Fund processing", ctx.GasMeter().GasConsumed())

	res, gasLimit, err := k.callEVM(
		ctx,
		sender,
		contract,
		msg.Data,
		fundAmount,
		true,
	)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "failed to apply transaction")
	}

	fmt.Println("keeper, after evm execution", ctx.GasMeter().GasConsumed())

	defer func() {
		telemetry.IncrCounterWithLabels(
			[]string{"tx", "msg", "call_evm", "total"},
			1,
			labels,
		)

		if res.GasUsed != 0 {
			telemetry.IncrCounterWithLabels(
				[]string{"tx", "msg", "call_evm", "gas_used", "total"},
				float32(res.GasUsed),
				labels,
			)

			// Observe which users define a gas limit >> gas used. Note, that
			// gas_limit and gas_used are always > 0
			gasRatio, err := sdk.NewDec(int64(gasLimit)).QuoInt64(int64(res.GasUsed)).Float64()
			if err == nil {
				telemetry.SetGaugeWithLabels(
					[]string{"tx", "msg", "call_evm", "gas_limit", "per", "gas_used"},
					float32(gasRatio),
					labels,
				)
			}
		}
	}()

	attrs := []sdk.Attribute{
		sdk.NewAttribute(sdk.AttributeKeyAmount, fundAmount.String()),
		// add event for ethereum transaction hash format
		sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxHash, res.Hash),
		// add event for index of valid ethereum tx
		sdk.NewAttribute(evmtypes.AttributeKeyTxIndex, strconv.FormatUint(txIndex, 10)),
		// add event for eth tx gas used, we can't get it from cosmos tx result when it contains multiple eth tx msgs.
		sdk.NewAttribute(evmtypes.AttributeKeyTxGasUsed, strconv.FormatUint(res.GasUsed, 10)),
	}

	fmt.Println("keeper, after add attr1", ctx.GasMeter().GasConsumed())

	if len(ctx.TxBytes()) > 0 {
		// add event for tendermint transaction hash format
		hash := tmbytes.HexBytes(tmtypes.Tx(ctx.TxBytes()).Hash())
		attrs = append(attrs, sdk.NewAttribute(evmtypes.AttributeKeyTxHash, hash.String()))
	}

	fmt.Println("keeper, after add attr2", ctx.GasMeter().GasConsumed())

	if contract != nil {
		attrs = append(attrs, sdk.NewAttribute(evmtypes.AttributeKeyRecipient, contract.Hex()))
	}

	fmt.Println("keeper, after add contract res", ctx.GasMeter().GasConsumed())

	if res.Failed() {
		attrs = append(attrs, sdk.NewAttribute(evmtypes.AttributeKeyEthereumTxFailed, res.VmError))
	}

	txLogAttrs := make([]sdk.Attribute, len(res.Logs))
	for i, log := range res.Logs {
		value, err := json.Marshal(log)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "failed to encode log")
		}
		txLogAttrs[i] = sdk.NewAttribute(evmtypes.AttributeKeyTxLog, string(value))
	}

	// emit events
	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgCallEVM,
			attrs...,
		),
		sdk.NewEvent(
			evmtypes.EventTypeTxLog,
			txLogAttrs...,
		),
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, evmtypes.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
		),
	})

	fmt.Println("keeper, after after event emission", ctx.GasMeter().GasConsumed())

	return res, nil
}
