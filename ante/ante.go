package ante

import (
	"fmt"
	"runtime/debug"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	tmlog "cosmossdk.io/log"
	//storetypes "cosmossdk.io/store/types"
	txsigning "cosmossdk.io/x/tx/signing"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	errortypes "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	evmante "github.com/xpladev/ethermint/app/ante"
	evmtypes "github.com/xpladev/ethermint/x/evm/types"

	volunteerante "github.com/xpladev/xpla/x/volunteer/ante"
)

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	Codec                  codec.BinaryCodec
	AccountKeeper          evmtypes.AccountKeeper
	BankKeeper             evmtypes.BankKeeper
	IBCKeeper              *ibckeeper.Keeper
	EvmKeeper              evmante.EVMKeeper
	FeegrantKeeper         authante.FeegrantKeeper
	VolunteerKeeper        volunteerante.VolunteerKeeper
	ExtensionOptionChecker authante.ExtensionOptionChecker
	SignModeHandler        *txsigning.HandlerMap
	SigGasConsumer         authante.SignatureVerificationGasConsumer
	FeeMarketKeeper        evmante.FeeMarketKeeper
	MaxTxGasWanted         uint64
	TxFeeChecker           authante.TxFeeChecker

	BypassMinFeeMsgTypes  []string
	TXCounterStoreService corestoretypes.KVStoreService
	WasmKeeper            *wasmkeeper.Keeper
	WasmConfig            *wasmtypes.WasmConfig
}

var disabledAuthzMsgs = []string{
	sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
	sdk.MsgTypeURL(&vestingtypes.MsgCreateVestingAccount{}),
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
				case "/ethermint.types.v1.ExtensionOptionsWeb3Tx":
					// handle as normal Cosmos SDK tx, except signature is checked for EIP712 representation
					anteHandler = newLegacyCosmosAnteHandlerEip712(opts)
				case "/ethermint.types.v1.ExtensionOptionDynamicFeeTx":
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
	var sigGasConsumer = opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = SigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		evmante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		// disable the Msg types that cannot be included on an authz.MsgExec msgs field
		evmante.NewAuthzLimiterDecorator(disabledAuthzMsgs),
		volunteerante.NewRejectDelegateVolunteerValidatorDecorator(opts.VolunteerKeeper),
		authante.NewSetUpContextDecorator(), // second decorator. SetUpContext must be called before other decorators
		authante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(opts.WasmKeeper.GetGasRegister()),
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
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}

func newEthAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEthSetUpContextDecorator(opts.EvmKeeper),                      // outermost AnteDecorator. SetUpContext must be called first
		evmante.NewEthMempoolFeeDecorator(opts.EvmKeeper),                        // Check eth effective gas price against minimal-gas-prices
		evmante.NewEthMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper), // Check eth effective gas price against the global MinGasPrice
		evmante.NewEthValidateBasicDecorator(opts.EvmKeeper),
		evmante.NewEthSigVerificationDecorator(opts.EvmKeeper),
		evmante.NewEthAccountVerificationDecorator(opts.AccountKeeper, opts.EvmKeeper),
		evmante.NewEthGasConsumeDecorator(opts.EvmKeeper, opts.MaxTxGasWanted),
		evmante.NewCanTransferDecorator(opts.EvmKeeper),
		evmante.NewEthIncrementSenderSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator.
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
		evmante.NewEthEmitEventDecorator(opts.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newLegacyCosmosAnteHandlerEip712(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		// disable the Msg types that cannot be included on an authz.MsgExec msgs field
		evmante.NewAuthzLimiterDecorator(disabledAuthzMsgs),
		volunteerante.NewRejectDelegateVolunteerValidatorDecorator(opts.VolunteerKeeper),
		authante.NewSetUpContextDecorator(),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper, opts.BypassMinFeeMsgTypes), // Check eth effective gas price against the global MinGasPrice
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(opts.AccountKeeper),
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		// Note: signature verification uses EIP instead of the cosmos signature validator
		//nolint: staticcheck
		evmante.NewLegacyEip712SigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
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
