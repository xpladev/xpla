package ante

import (
	"fmt"
	"runtime/debug"

	errorsmod "cosmossdk.io/errors"
	tmlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authante "github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	sdkvestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	ibcante "github.com/cosmos/ibc-go/v7/modules/core/ante"
	ibckeeper "github.com/cosmos/ibc-go/v7/modules/core/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	cosmosante "github.com/evmos/evmos/v14/app/ante/cosmos"
	evmante "github.com/evmos/evmos/v14/app/ante/evm"
	anteutils "github.com/evmos/evmos/v14/app/ante/utils"
	evmtypes "github.com/evmos/evmos/v14/x/evm/types"
	vestingtypes "github.com/evmos/evmos/v14/x/vesting/types"

	volunteerante "github.com/xpladev/xpla/x/volunteer/ante"
)

// HandlerOptions extend the SDK's AnteHandler opts by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	Cdc                codec.BinaryCodec
	AccountKeeper      evmtypes.AccountKeeper
	BankKeeper         evmtypes.BankKeeper
	DistributionKeeper anteutils.DistributionKeeper
	IBCKeeper          *ibckeeper.Keeper
	StakingKeeper      vestingtypes.StakingKeeper
	EvmKeeper          evmante.EVMKeeper
	FeegrantKeeper     authante.FeegrantKeeper
	VolunteerKeeper    volunteerante.VolunteerKeeper
	SignModeHandler    authsigning.SignModeHandler
	SigGasConsumer     authante.SignatureVerificationGasConsumer
	FeeMarketKeeper    evmante.FeeMarketKeeper
	MaxTxGasWanted     uint64
	TxFeeChecker       authante.TxFeeChecker

	BypassMinFeeMsgTypes []string
	TxCounterStoreKey    storetypes.StoreKey
	WasmConfig           wasmTypes.WasmConfig
}

// NewAnteHandler returns an 'AnteHandler' that will run actions before a tx is sent to a module's handler.
func NewAnteHandler(opts HandlerOptions) (sdk.AnteHandler, error) {
	if opts.AccountKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "account keeper is required for AnteHandler")
	}
	if opts.BankKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "bank keeper is required for AnteHandler")
	}
	if opts.StakingKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "staking keeper is required for AnteHandler")
	}
	if opts.SignModeHandler == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "sign mode handler is required for ante builder")
	}
	if opts.IBCKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "IBC keeper is required for AnteHandler")
	}
	if opts.EvmKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "EVM keeper is required for AnteHandler")
	}
	if opts.FeegrantKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "Feegrant keeper is required for AnteHandler")
	}
	if opts.FeeMarketKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "Feemarket keeper is required for AnteHandler")
	}
	if opts.VolunteerKeeper == nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrLogic, "staking keeper is required for AnteHandler")
	}

	sigGasConsumer := opts.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = authante.DefaultSigVerificationGasConsumer
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
			return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
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
		cosmosante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		cosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvestingtypes.MsgCreateVestingAccount{}),
		),
		volunteerante.NewRejectDelegateVolunteerValidatorDecorator(opts.VolunteerKeeper),
		authante.NewSetUpContextDecorator(), // second decorator. SetUpContext must be called before other decorators
		wasmkeeper.NewLimitSimulationGasDecorator(opts.WasmConfig.SimulationGasLimit),
		wasmkeeper.NewCountTXDecorator(opts.TxCounterStoreKey),
		NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper, opts.BypassMinFeeMsgTypes),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		cosmosante.NewVestingDelegationDecorator(opts.AccountKeeper, opts.StakingKeeper, opts.BankKeeper, opts.Cdc),
		authante.NewSetPubKeyDecorator(opts.AccountKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, sigGasConsumer),
		authante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
	}
	return sdk.ChainAnteDecorators(anteDecorators...)
}

func newEthAnteHandler(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEthSetUpContextDecorator(opts.EvmKeeper),                                      // outermost AnteDecorator. SetUpContext must be called first
		NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper, opts.BypassMinFeeMsgTypes), // Check eth effective gas price against the global MinGasPrice
		evmante.NewEthValidateBasicDecorator(opts.EvmKeeper),
		evmante.NewEthSigVerificationDecorator(opts.EvmKeeper),
		evmante.NewEthAccountVerificationDecorator(opts.AccountKeeper, opts.EvmKeeper),
		evmante.NewCanTransferDecorator(opts.EvmKeeper),
		evmante.NewEthVestingTransactionDecorator(opts.AccountKeeper, opts.BankKeeper, opts.EvmKeeper),
		evmante.NewEthGasConsumeDecorator(opts.BankKeeper, opts.DistributionKeeper, opts.EvmKeeper, opts.StakingKeeper, opts.MaxTxGasWanted),
		evmante.NewEthIncrementSenderSequenceDecorator(opts.AccountKeeper), // innermost AnteDecorator.
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
		evmante.NewEthEmitEventDecorator(opts.EvmKeeper), // emit eth tx hash and index at the very last ante handler.
	)
}

// newCosmosAnteHandlerEip712 creates the ante handler for transactions signed with EIP712
func newLegacyCosmosAnteHandlerEip712(opts HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		cosmosante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		cosmosante.NewAuthzLimiterDecorator( // disable the Msg types that cannot be included on an authz.MsgExec msgs field
			sdk.MsgTypeURL(&evmtypes.MsgEthereumTx{}),
			sdk.MsgTypeURL(&sdkvestingtypes.MsgCreateVestingAccount{}),
		),
		volunteerante.NewRejectDelegateVolunteerValidatorDecorator(opts.VolunteerKeeper),
		authante.NewSetUpContextDecorator(),
		authante.NewValidateBasicDecorator(),
		authante.NewTxTimeoutHeightDecorator(),
		NewMinGasPriceDecorator(opts.FeeMarketKeeper, opts.EvmKeeper, opts.BypassMinFeeMsgTypes), // Check eth effective gas price against the global MinGasPrice
		authante.NewValidateMemoDecorator(opts.AccountKeeper),
		authante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		authante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		cosmosante.NewVestingDelegationDecorator(opts.AccountKeeper, opts.StakingKeeper, opts.BankKeeper, opts.Cdc),
		// SetPubKeyDecorator must be called before all signature verification decorators
		authante.NewSetPubKeyDecorator(opts.AccountKeeper),
		authante.NewValidateSigCountDecorator(opts.AccountKeeper),
		authante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		// Note: signature verification uses EIP instead of the cosmos signature validator
		//nolint: staticcheck
		cosmosante.NewLegacyEip712SigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		authante.NewIncrementSequenceDecorator(opts.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(opts.IBCKeeper),
		evmante.NewGasWantedDecorator(opts.EvmKeeper, opts.FeeMarketKeeper),
	)
}

func Recover(logger tmlog.Logger, err *error) {
	if r := recover(); r != nil {
		*err = errorsmod.Wrapf(sdkerrors.ErrPanic, "%v", r)

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
