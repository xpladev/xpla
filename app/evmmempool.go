package app

import (
	"fmt"

	"cosmossdk.io/log/v2"

	"github.com/cosmos/cosmos-sdk/baseapp"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkmempool "github.com/cosmos/cosmos-sdk/types/mempool"

	evmmempool "github.com/cosmos/evm/mempool"
	evmserver "github.com/cosmos/evm/server"
	evmtypes "github.com/cosmos/evm/x/vm/types"
)

// configureEVMMempool sets up the EVM mempool and related handlers using viper configuration.
func (app *XplaApp) configureEVMMempool(appOpts servertypes.AppOptions, logger log.Logger) error {
	if evmtypes.GetChainConfig() == nil {
		logger.Debug("evm chain config is not set, skipping mempool configuration")
		return nil
	}

	cosmosPoolMaxTx := evmserver.GetCosmosPoolMaxTx(appOpts, logger)
	if cosmosPoolMaxTx < 0 {
		// XPLA runs Cosmos EVM's app-side mempool as the required handler for
		// CometBFT's app mempool. Disabling it would leave InsertTx/ReapTxs
		// without handlers, so fail during startup instead of falling back.
		return fmt.Errorf("mempool.max-txs=%d is unsupported: XPLA requires the Cosmos EVM app-side mempool", cosmosPoolMaxTx)
	}

	mempoolConfig, err := app.createMempoolConfig(appOpts, logger)
	if err != nil {
		return fmt.Errorf("failed to get mempool config: %w", err)
	}
	if err := evmserver.ValidateReapBounds(appOpts, mempoolConfig.BlockGasLimit); err != nil {
		return err
	}

	txEncoder := evmmempool.NewTxEncoder(app.txConfig)
	evmRechecker := evmmempool.NewTxRechecker(app.GetAnteHandler(), txEncoder)
	cosmosRechecker := evmmempool.NewTxRechecker(app.GetAnteHandler(), txEncoder)
	evmMempool := evmmempool.NewMempool(
		app.CreateQueryContext,
		logger,
		app.EvmKeeper,
		app.FeeMarketKeeper,
		app.txConfig,
		evmRechecker,
		cosmosRechecker,
		mempoolConfig,
		cosmosPoolMaxTx,
	)
	app.EVMMempool = evmMempool
	app.SetMempool(evmMempool)
	checkTxHandler := evmMempool.NewCheckTxHandler(app.txConfig.TxDecoder(), evmserver.GetMempoolCheckTxTimeout(appOpts, logger))
	app.SetCheckTxHandler(checkTxHandler)
	app.SetInsertTxHandler(evmMempool.NewInsertTxHandler(app.TxDecode))
	app.SetReapTxsHandler(evmMempool.NewReapTxsHandler())

	abciProposalHandler := baseapp.NewDefaultProposalHandler(evmMempool, app)
	abciProposalHandler.SetSignerExtractionAdapter(
		evmmempool.NewEthSignerExtractionAdapter(
			sdkmempool.NewDefaultSignerExtractionAdapter(),
		),
	)
	app.SetPrepareProposal(abciProposalHandler.PrepareProposalHandler())
	app.SetProcessProposal(abciProposalHandler.ProcessProposalHandler())
	app.SetPrepareCheckStater(func(_ sdk.Context) {
		if !evmMempool.HasEventBus() {
			evmMempool.NotifyNewBlock()
		}
	})

	return nil
}

// createMempoolConfig creates a new EVM mempool config with the default configuration
// and overrides it with values from appOpts if they exist and are non-zero.
func (app *XplaApp) createMempoolConfig(appOpts servertypes.AppOptions, logger log.Logger) (*evmmempool.Config, error) {
	return evmserver.ResolveMempoolConfig(app.GetAnteHandler(), appOpts, logger), nil
}
