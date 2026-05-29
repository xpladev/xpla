package multichain_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/cosmos/interchaintest/v11/ibc"

	"github.com/stretchr/testify/assert"

	"github.com/xpladev/xpla/tests/e2e/multichain"
)

const (
	Sha256SkipEntryPoint = "4ee07a1474cb1429cfbdba98fb52ca2efc2fe8602f8e1978dbc3f45b71511ca9"
	SaltHex              = "74657374696e67" // "testing" hex encoded
)

func TestIbcCallbacks(t *testing.T) {
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

	// prepare contract environment
	entryPointID, err := xplaChain.StoreContract(ctx, xplaUser.KeyName(), "../misc/skip_go_entry_point.wasm")
	assert.NoError(t, err)
	adapterID, err := xplaChain.StoreContract(ctx, xplaUser.KeyName(), "../misc/skip_go_ibc_adapter_ibc_callbacks.wasm")
	assert.NoError(t, err)

	predictedEntryPointAddr, err := multichain.BuildAddr(ctx, xplaChain.GetFullNode(), Sha256SkipEntryPoint, xplaUser.FormattedAddress(), SaltHex)
	assert.NoError(t, err)
	instantiateAdapterJSON := fmt.Sprintf(`{"entry_point_contract_address":"%s"}`, predictedEntryPointAddr)
	adapterAddr, err := xplaChain.InstantiateContract(ctx, xplaUser.KeyName(), adapterID, instantiateAdapterJSON, true, "--gas", "auto", "--gas-adjustment", "1.5")
	assert.NoError(t, err)

	instantiateEntrypointJSON := fmt.Sprintf(`{"swap_venues":[], "ibc_transfer_contract_address": "%s"}`, adapterAddr)
	entryPointAddr, err := multichain.InstantiateContract2(ctx, xplaChain.GetFullNode(), xplaUser.KeyName(), entryPointID, instantiateEntrypointJSON, SaltHex, true, "--gas", "auto", "--gas-adjustment", "1.5")
	assert.Equal(t, predictedEntryPointAddr, entryPointAddr)
	assert.NoError(t, err)

	// generate dest callback memo
	str := "transfer/channel-0/stake"
	h := sha256.New()
	h.Write([]byte(str))
	bs := h.Sum(nil)

	recipientDenom := fmt.Sprintf("ibc/%X", bs)

	ibcHooksData := fmt.Sprintf(`"wasm": {
						"contract": "%s",
						"msg": {
						  "action": {
							"sent_asset": {
							  "native": {
								"denom":"%s",
								"amount":"1"
							  }
							},
							"exact_out": false,
							"timeout_timestamp": %d,
							"action": {
							  "transfer":{
								"to_address": "%s"
							  }
							}
						  }
						}
					  }`, entryPointAddr, recipientDenom, time.Now().Add(time.Minute).UnixNano(), xplaUser.FormattedAddress())
	destCallbackData := fmt.Sprintf(`"dest_callback": {
					"address": "%s",
					"gas_limit": "%d"
				  }`, adapterAddr, 10_000_000)
	memoStr := fmt.Sprintf("{%s, %s}", destCallbackData, ibcHooksData)

	// send ibc with memo string
	walletAmount := ibc.WalletAmount{
		Address: adapterAddr,
		Denom:   simdChain.Config().Denom,
		Amount:  sdkmath.OneInt(),
	}
	tx, err := simdChain.SendIBCTransfer(ctx, simdChannel.ChannelID, simdUser.FormattedAddress(), walletAmount, ibc.TransferOptions{Memo: memoStr})
	assert.NoError(t, err)
	assert.NoError(t, tx.Validate())

	// Flush the relayer
	err = ibcSetup.FlushRelayer(ctx)
	assert.NoError(t, err)

	// check local account balance
	coins, err := xplaChain.BankQueryAllBalances(ctx, xplaUser.FormattedAddress())
	assert.NoError(t, err)
	for _, coin := range coins {
		if coin.Denom != xplaChain.Config().Denom {
			assert.Equal(t, sdkmath.OneInt(), coin.Amount)
		}
	}
}
