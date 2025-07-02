package ante

import (
	"fmt"
	"runtime/debug"

	ibcante "github.com/cosmos/ibc-go/v10/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v10/modules/core/keeper"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	tmlog "cosmossdk.io/log"

	txsigning "cosmossdk.io/x/tx/signing"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"

	cosmosante "github.com/cosmos/evm/ante/cosmos"
	evmante "github.com/cosmos/evm/ante/evm"
	evmanteinterfaces "github.com/cosmos/evm/ante/interfaces"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	volunteerante "github.com/xpladev/xpla/x/volunteer/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ExtensionOptionChecker authante.ExtensionOptionChecker
	FeegrantKeeper         authante.FeegrantKeeper
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         authante.SignatureVerificationGasConsumer

	AccountKeeper         evmtypes.AccountKeeper
	BankKeeper            evmtypes.BankKeeper
	Codec                 codec.BinaryCodec
	IBCKeeper             *ibckeeper.Keeper
	EvmKeeper             evmanteinterfaces.EVMKeeper
	VolunteerKeeper       volunteerante.VolunteerKeeper
	BypassMinFeeMsgTypes  []string
	FeeMarketKeeper       evmanteinterfaces.FeeMarketKeeper
	MaxTxGasWanted        uint64
	TxFeeChecker          authante.TxFeeChecker
	TXCounterStoreService corestoretypes.KVStoreService
	WasmConfig            *wasmtypes.NodeConfig
}

// NewAnteHandler returns an 'AnteHandler' that will run actions before a tx is sent to a module's handler.
func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "sign mode handler is required for AnteHandler")
	}
	if opts.IBCKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "IBC keeper is required for AnteHandler")
	}
	if opts.EvmKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "EVM keeper is required for AnteHandler")
	}
	if opts.FeegrantKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "Feegrant keeper is required for AnteHandler")
	}
	if opts.FeeMarketKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "FeeMarket keeper is required for AnteHandler")
	}
	if opts.VolunteerKeeper == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if opts.Codec == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "codec is required for AnteHandler")
	}
	if opts.SigGasConsumer == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "signature gas consumer is required for AnteHandler")
	}
	if opts.TxFeeChecker == nil {
		return nil, errorsmod.Wrap(errortypes.ErrLogic, "tx fee checker is required for AnteHandler")
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
				case "/cosmos.evm.vm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = newEthAnteHandler(opts)
				case "/cosmos.evm.types.v1.ExtensionOptionDynamicFeeTx":
					// cosmos-sdk tx with dynamic fee extension
					anteHandler = newCosmosAnteHandler(opts)
				default:
					return ctx, errorsmod.Wrapf(
						errortypes.ErrUnknownExtensionOptions,
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
			return ctx, errorsmod.Wrapf(errortypes.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}, nil
}

func newCosmosAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = SigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		cosmosante.NewRejectMessagesDecorator(), // reject MsgEthereumTxs
		// disable the Msg types that cannot be included on an authz.MsgExec msgs field
		cosmosante.NewAuthzLimiterDecorator(
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
		),
		volunteerante.NewRejectDelegateVolunteerValidatorDecorator(opts.VolunteerKeeper),
		authante.NewSetUpContextDecorator(), // second decorator. SetUpContext must be called before other decorators
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TXCounterStoreService),
		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper, opts.BypassMinFeeMsgTypes),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		authante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}

func newEthAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEVMMonoDecorator(
			opts.AccountKeeper,
			opts.FeeMarketKeeper,
			opts.EvmKeeper,
			opts.MaxTxGasWanted,
		),
	)
}

func Recover(logger tmlog.Logger, err *error) {
	if r := recover(); r != nil {
		*err = errorsmod.Wrapf(errortypes.ErrPanic, "%v", r)

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
