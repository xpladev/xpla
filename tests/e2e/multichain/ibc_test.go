package multichain_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"

	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/assert"

	"github.com/xpladev/xpla/tests/e2e/multichain"
)

func TestIbcMsgTransfer(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	ibcSetup := multichain.StartXplaChainAndSimdWithIBC(t, ctx, multichain.LocalImage)

	// Get the first user
	xplaUser := ibcSetup.XplaUsers[0]
	simdUser := ibcSetup.SimdUsers[0]

	// Get the channel
	simdChannels, err := ibcSetup.GetSimdChannels(ctx)
	assert.NoError(t, err)
	simdChannel := simdChannels[0]

	// Get the chain
	xplaChain := ibcSetup.XplaChain
	simdChain := ibcSetup.SimdChain

	// call contract from simd
	walletAmount := ibc.WalletAmount{
		Address: xplaUser.FormattedAddress(),
		Denom:   simdChain.Config().Denom,
		Amount:  sdkmath.OneInt(),
	}
	tx, err := simdChain.SendIBCTransfer(ctx, simdChannel.ChannelID, simdUser.FormattedAddress(), walletAmount, ibc.TransferOptions{})

	assert.NoError(t, err)
	assert.NoError(t, tx.Validate())

	// Flush the relayer
	err = ibcSetup.FlushRelayer(ctx)
	assert.NoError(t, err)

	coins, err := xplaChain.BankQueryAllBalances(ctx, xplaUser.FormattedAddress())
	assert.NoError(t, err)
	for _, coin := range coins {
		if coin.Denom != xplaChain.Config().Denom {
			assert.Equal(t, sdkmath.OneInt(), coin.Amount)
		}
	}
}
