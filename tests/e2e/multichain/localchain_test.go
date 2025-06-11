package multichain_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"

	"github.com/xpladev/xpla/tests/e2e/multichain"
)

func TestLocalChain(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}
	t.Parallel()

	ctx := context.Background()

	chains, _ := multichain.XplaChainStartWithIBC(t, ctx, multichain.LocalImage)

	xplaChain := chains[0].(*cosmos.CosmosChain)
	supply, err := xplaChain.BankQueryTotalSupplyOf(ctx, "axpla")
	require.NoError(t, err)

	assert.Equal(t, "110000000000000000000000000axpla", supply.String())
}
