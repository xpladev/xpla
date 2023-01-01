package ante

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	ibcante "github.com/cosmos/ibc-go/v3/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	evmante "github.com/evmos/ethermint/app/ante"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// HandlerOptions extend the SDK's AnteHandler opts by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	AccountKeeper   evmtypes.AccountKeeper
	BankKeeper      evmtypes.BankKeeper
	IBCKeeper       *ibckeeper.Keeper
	EvmKeeper       evmante.EVMKeeper
	FeegrantKeeper  authante.FeegrantKeeper
	SignModeHandler authsigning.SignModeHandler
	SigGasConsumer  authante.SignatureVerificationGasConsumer
	FeeMarketKeeper evmtypes.FeeMarketKeeper
	MaxTxGasWanted  uint64

	BypassMinFeeMsgTypes []string
	TxCounterStoreKey    sdk.StoreKey
	WasmConfig           wasmTypes.WasmConfig
}

// NewAnteHandler returns an 'AnteHandler' that will run actions before a tx is sent to a module's handler.
func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}

	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		defer Recover(ctx.Logger(), &err)

		txWithExtensions, ok := tx.(authante.HasExtensionOptionsTx)
		if ok {
			eopts := txWithExtensions.GetExtensionOptions()
			if len(eopts) > 0 {
				switch typeURL := eopts[0].GetTypeUrl(); typeURL {
				case "/ethermint.evm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = newEthAnteHandler(opts)
				default:
					return ctx, sdkerrors.Wrapf(
						sdkerrors.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx
		switch tx.(type) {
		case sdk.Tx:
			anteHandler = newCosmosAnteHandler(opts)
		default:
			return ctx, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}, nil
}

func newCosmosAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	var sigGasConsumer = opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = SigVerificationGasConsumer
	}

	fmt.Println("Cosm ante handler")

	anteDecorators := []sdk.AnteDecorator{
		evmante.RejectMessagesDecorator{},   // reject MsgEthereumTxs
		authante.NewSetUpContextDecorator(), // second decorator. SetUpContext must be called before other decorators
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		NewGasfeeCheckDecorator("Limit simulator"),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		NewGasfeeCheckDecorator("Count tx"),
		authante.NewRejectExtensionOptionsDecorator(),
		NewGasfeeCheckDecorator("RejectExtentionOption"),
		NewMempoolFeeDecorator(opts.BypassMinFeeMsgTypes),
		NewGasfeeCheckDecorator("MempoolFee"),
		evmante.NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper),
		NewGasfeeCheckDecorator("Min gas fee"),
		authante.NewValidateBasicDecorator(),
		NewGasfeeCheckDecorator("ValidateBasic"),
		authante.NewTxTimeoutHeightDecorator(),
		NewGasfeeCheckDecorator("Tx timeout height"),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		NewGasfeeCheckDecorator("ValidateMemo"),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		NewGasfeeCheckDecorator("Consume Gas for tx size"),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper),
		NewGasfeeCheckDecorator("Deduct Fee"),
		authante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		NewGasfeeCheckDecorator("Set pub key"),
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		NewGasfeeCheckDecorator("Validator sig count"),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		NewGasfeeCheckDecorator("Sig gas consume"),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		NewGasfeeCheckDecorator("Gas fee check 2"),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator
		NewGasfeeCheckDecorator("Inc fee check"),
		ibcante.NewAnteDecorator(opts.IBCKeeper),
		NewGasfeeCheckDecorator("Ibc ante fee check"),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
		NewGasfeeCheckDecorator("Gas fee check 3"),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}

func newEthAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	fmt.Println("EVM ante handler")

	return sdk.ChainAnteDecorators(
		evmante.NewEthSetUpContextDecorator(opts.EvmKeeper), // outermost AnteDecorator. SetUpContext must be called first
		NewGasfeeCheckDecorator("SetUpContext"),
		evmante.NewEthMempoolFeeDecorator(opts.EvmKeeper), // Check eth effective gas price against minimal-gas-prices
		NewGasfeeCheckDecorator("MempoolFee"),
		evmante.NewEthMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
		NewGasfeeCheckDecorator("MinGasPrice"),
		evmante.NewEthValidateBasicDecorator(opts.EvmKeeper),
		NewGasfeeCheckDecorator("ValidateBasic"),
		evmante.NewEthSigVerificationDecorator(opts.EvmKeeper),
		NewGasfeeCheckDecorator("SigVerification"),
		evmante.NewEthAccountVerificationDecorator(opts.AccountKeeper, opts.EvmKeeper),
		NewGasfeeCheckDecorator("AccVerification"),
		evmante.NewEthGasConsumeDecorator(opts.EvmKeeper, opts.MaxTxGasWanted),
		NewGasfeeCheckDecorator("GasConsume"),
		evmante.NewCanTransferDecorator(opts.EvmKeeper),
		NewGasfeeCheckDecorator("CanTransfer"),
		evmante.NewEthGasConsumeDecorator(opts.EvmKeeper, opts.MaxTxGasWanted),
		NewGasfeeCheckDecorator("GasConsume 2"),
		evmante.NewEthIncrementSenderSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator.
		NewGasfeeCheckDecorator("Increment sender seq"),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
		NewGasfeeCheckDecorator("GasWanted"),
		evmante.NewEthEmitEventDecorator(opts.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
		NewGasfeeCheckDecorator("Event emission"),
	)
}

func Recover(logger tmlog.Logger, err *error) {
	if r := recover(); r != nil {
		*err = sdkerrors.Wrapf(sdkerrors.ErrPanic, "%v", r)

		if e, ok := r.(error); ok {
			logger.Error(
				"ante handler panicked",
				"error", e,
				"stack trace", string(debug.Stack()),
			)
		} else {
			logger.Error(
				"ante handler panicked",
				"recover", fmt.Sprintf("%v", r),
			)
		}
	}
}
