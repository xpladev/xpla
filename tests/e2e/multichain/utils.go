package multichain

import (
	"context"
	"testing"

	"github.com/docker/docker/client"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/strangelove-ventures/interchaintest/v8/testreporter"

	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/assert"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	wallet  = "xpla"
	denom   = "axpla"
	display = "XPLA"

	minGasPrice = "280000000000"
)

var (
	decimals = int64(18)

	LocalImage = []ibc.DockerImage{
		{
			Repository: "xpla",
			Version:    "local",
			UIDGID:     "1025:1025",
		},
	}
)

func XplaChainSpec(
	numValidators int,
	numFullNodes int,
	chainID string,
	version []ibc.DockerImage,
) *interchaintest.ChainSpec {

	genesis := []cosmos.GenesisKV{
		cosmos.NewGenesisKV("app_state.gov.params.voting_period", "1m"),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.denom", denom),
		cosmos.NewGenesisKV("app_state.gov.params.min_deposit.0.amount", "1"),
		cosmos.NewGenesisKV("app_state.gov.params.expedited_min_deposit.0.denom", denom),
		cosmos.NewGenesisKV("app_state.gov.params.expedited_min_deposit.0.amount", "1"),

		cosmos.NewGenesisKV("app_state.mint.params.mint_denom", denom),
		cosmos.NewGenesisKV("app_state.mint.minter.inflation", "0.000000000000000000"),
		cosmos.NewGenesisKV("app_state.mint.params.inflation_rate_change", "0.000000000000000000"),
		cosmos.NewGenesisKV("app_state.mint.params.inflation_min", "0.000000000000000000"),
		cosmos.NewGenesisKV("app_state.mint.params.inflation_max", "0.000000000000000000"),

		cosmos.NewGenesisKV("consensus.params.block.max_gas", "50000000000"),
		cosmos.NewGenesisKV("app_state.evm.params.evm_denom", denom),
		cosmos.NewGenesisKV("app_state.feemarket.params.min_gas_price", minGasPrice),

		cosmos.NewGenesisKV("app_state.crisis.constant_fee.denom", denom),
	}

	return &interchaintest.ChainSpec{
		Name:          "xpla",
		NumValidators: &numValidators,
		NumFullNodes:  &numFullNodes,
		ChainConfig: ibc.ChainConfig{
			Name:           "xpla_1-1",
			Type:           "cosmos",
			ChainID:        chainID,
			Images:         version,
			Bin:            "xplad",
			Bech32Prefix:   wallet,
			Denom:          denom,
			CoinType:       "60",
			GasPrices:      minGasPrice + denom,
			GasAdjustment:  1.5,
			TrustingPeriod: "168h0m0s",
			ModifyGenesis:  cosmos.ModifyGenesis(genesis),
			CoinDecimals:   &decimals,
			// open the port for the EVM on all nodes
			ExposeAdditionalPorts: []string{"8545/tcp"},
		},
	}
}

func XplaChainStart(t *testing.T, ctx context.Context, version []ibc.DockerImage) (*cosmos.CosmosChain, *client.Client) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xpla", "xplapub")

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	client, network := interchaintest.DockerSetup(t)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		XplaChainSpec(1, 0, "interchaintest_1-1", version),
	})
	chains, err := cf.Chains(t.Name())
	assert.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)

	ic := interchaintest.NewInterchain().AddChain(chain)
	err = ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	assert.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	return chain, client
}

func XplaChainStartWithIBC(t *testing.T, ctx context.Context, version []ibc.DockerImage) ([]ibc.Chain, *client.Client) {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("xpla", "xplapub")

	rep := testreporter.NewNopReporter()
	eRep := rep.RelayerExecReporter(t)

	client, network := interchaintest.DockerSetup(t)

	numValidators := 1
	numFullNodes := 0

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		XplaChainSpec(1, 0, "xpla_1-1", version),
		{
			Name:    "ibc-go-simd",
			Version: "v8.7.0",
			ChainConfig: ibc.ChainConfig{
				Name:           "ibc-go-simd",
				Type:           "cosmos",
				ChainID:        "ibc-go-simd-2",
				Bin:            "simd",
				Bech32Prefix:   "cosmos",
				Denom:          "stake",
				CoinType:       "118",
				GasPrices:      "0.00stake",
				GasAdjustment:  1.5,
				TrustingPeriod: "168h0m0s",
			},
			NumValidators: &numValidators,
			NumFullNodes:  &numFullNodes,
		},
	})
	chains, err := cf.Chains(t.Name())
	assert.NoError(t, err)

	chain := chains[0].(*cosmos.CosmosChain)
	ibcSimd := chains[1].(*cosmos.CosmosChain)

	rf := interchaintest.NewBuiltinRelayerFactory(ibc.CosmosRly, zaptest.NewLogger(t))
	r := rf.Build(t, client, network)

	ibcPathName := "path"
	ic := interchaintest.NewInterchain().
		AddChain(chain).
		AddChain(ibcSimd).
		AddRelayer(r, "relayer").
		AddLink(interchaintest.InterchainLink{
			Chain1:  chain,
			Chain2:  ibcSimd,
			Relayer: r,
			Path:    ibcPathName,
		})

	assert.NoError(t, ic.Build(ctx, eRep, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: false,
	}))
	t.Cleanup(func() {
		_ = ic.Close()
	})

	return chains, client
}
