package ante

import (
	"fmt"
	"runtime/debug"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ibcante "github.com/cosmos/ibc-go/v3/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
	evmante "github.com/tharsis/ethermint/app/ante"
	evmtypes "github.com/tharsis/ethermint/x/evm/types"
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
	if opts.EvmKeeper == nil {
		return nil, sdkerrors.Wrap(sdkerrors.ErrLogic, "evm keeper is required for AnteHandler")
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
		sigGasConsumer = authante.DefaultSigVerificationGasConsumer
	}

	anteDecorators := []sdk.AnteDecorator{
		evmante.RejectMessagesDecorator{},   // reject MsgEthereumTxs
		authante.NewSetUpContextDecorator(), // second decorator. SetUpContext must be called before other decorators
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		authante.NewRejectExtensionOptionsDecorator(),
		//NewEvmMinGasFilter(opts.EvmKeeper), // filter out evm denom from min-gas-prices
		NewMempoolFeeDecorator(opts.BypassMinFeeMsgTypes),
		NewVestingAccountDecorator(),
		NewAuthzLimiterDecorator(
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&vesting.MsgCreateVestingAccount{}),
		),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper),
		authante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator
		ibcante.NewAnteDecorator(opts.IBCKeeper),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}

func newEthAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEthSetUpContextDecorator(opts.EvmKeeper), // outermost AnteDecorator. SetUpContext must be called first
		evmante.NewEthMempoolFeeDecorator(opts.EvmKeeper),   // Check eth effective gas price against minimal-gas-prices
		evmante.NewEthValidateBasicDecorator(opts.EvmKeeper),
		evmante.NewEthSigVerificationDecorator(opts.EvmKeeper),
		evmante.NewEthAccountVerificationDecorator(opts.AccountKeeper, opts.BankKeeper, opts.EvmKeeper),
		evmante.NewEthGasConsumeDecorator(opts.EvmKeeper, opts.MaxTxGasWanted),
		evmante.NewCanTransferDecorator(opts.EvmKeeper),
		evmante.NewEthIncrementSenderSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator.
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
