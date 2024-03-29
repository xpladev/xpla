package helpers

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	tmtypes "github.com/tendermint/tendermint/types"
	dbm "github.com/tendermint/tm-db"

	"github.com/xpladev/xpla/app"
	xplaapp "github.com/xpladev/xpla/app"
)

// SimAppChainID hardcoded chainID for simulation
const (
	SimAppChainID = "xpla-app"
)

// DefaultConsensusParams defines the default Tendermint consensus params used
// in XplaApp testing.
var DefaultConsensusParams = &abci.ConsensusParams{
	Block: &abci.BlockParams{
		MaxBytes: 200000,
		MaxGas:   2000000,
	},
	Evidence: &tmproto.EvidenceParams{
		MaxAgeNumBlocks: 302400,
		MaxAgeDuration:  504 * time.Hour, // 3 weeks is the max duration
		MaxBytes:        10000,
	},
	Validator: &tmproto.ValidatorParams{
		PubKeyTypes: []string{
			tmtypes.ABCIPubKeyTypeEd25519,
		},
	},
}

type EmptyAppOptions struct{}

func (EmptyAppOptions) Get(o string) interface{} { return nil }

func Setup(t *testing.T, chainId string, isCheckTx bool, invCheckPeriod uint) *xplaapp.XplaApp {
	t.Helper()

	app, genesisState := setup(!isCheckTx, invCheckPeriod)
	if !isCheckTx {
		// InitChain must be called to stop deliverState from being nil
		stateBytes, err := json.MarshalIndent(genesisState, "", " ")
		require.NoError(t, err)

		// Initialize the chain
		app.InitChain(
			abci.RequestInitChain{
				ChainId:         chainId,
				Validators:      []abci.ValidatorUpdate{},
				ConsensusParams: DefaultConsensusParams,
				AppStateBytes:   stateBytes,
			},
		)
	}

	return app
}

func setup(withGenesis bool, invCheckPeriod uint) (*xplaapp.XplaApp, xplaapp.GenesisState) {
	db := dbm.NewMemDB()
	encCdc := xplaapp.MakeTestEncodingConfig()
	app := xplaapp.NewXplaApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		xplaapp.DefaultNodeHome,
		invCheckPeriod,
		encCdc,
		app.GetEnabledProposals(),
		EmptyAppOptions{},
		[]wasm.Option{},
	)
	if withGenesis {
		return app, xplaapp.NewDefaultGenesisState()
	}

	return app, xplaapp.GenesisState{}
}
