package multichain_test

import (
	"context"
	"encoding/base64"
	"fmt"
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

func TestIbcHookWithIcs20(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	ibcSetup := multichain.StartXplaChainAndSimdWithIBC(t, ctx, multichain.LocalImage)

	// Get the first user
	xplaUser := ibcSetup.XplaUsers[0]

	// Get the chain
	xplaChain := ibcSetup.XplaChain

	// store cw20 contract
	cw20CodeID, err := xplaChain.StoreContract(ctx, xplaUser.KeyName(), "../misc/token.wasm")
	assert.NoError(t, err)

	cw20ContractAddress, err := xplaChain.InstantiateContract(ctx, xplaUser.KeyName(), cw20CodeID, fmt.Sprintf(`{"name":"testtoken","symbol":"TKN","decimals":6,"initial_balances":[{"address":"%s","amount":"100000000"}]}`, xplaUser.FormattedAddress()), true, "--gas", "auto", "--gas-adjustment", "1.5")
	assert.NoError(t, err)

	type BalanceResponse struct {
		Data struct {
			Balance string `json:"balance"`
		} `json:"data"`
	}

	balanceResponse := &BalanceResponse{}
	err = xplaChain.QueryContract(ctx, cw20ContractAddress, fmt.Sprintf(`{
		"balance": {
			"address": "%s"
		}
	}`, xplaUser.FormattedAddress()), balanceResponse)

	assert.NoError(t, err)
	assert.Equal(t, "100000000", balanceResponse.Data.Balance)

	// store cw20-ics20 contract
	cw20Ics20CodeID, err := xplaChain.StoreContract(ctx, xplaUser.KeyName(), "../misc/cw20_ics20.wasm")
	assert.NoError(t, err)

	cw20Ics20ContractAddress, err := xplaChain.InstantiateContract(ctx, xplaUser.KeyName(), cw20Ics20CodeID, fmt.Sprintf(`{"default_timeout": 1200, "gov_contract": "%s", "allowlist": [], "default_gas_limit": 200000}`, xplaUser.FormattedAddress()), true, "--gas", "auto", "--gas-adjustment", "1.5")
	assert.NoError(t, err)

	// create ics20 channel
	err = ibcSetup.CreateChannel(ctx, fmt.Sprintf("wasm.%s", cw20Ics20ContractAddress), "transfer", "ics20-1")
	assert.NoError(t, err)

	err = ibcSetup.FlushRelayer(ctx)
	assert.NoError(t, err)

	// send cw20 token to simd
	simdUser := ibcSetup.SimdUsers[0]
	simdChannels, err := ibcSetup.GetSimdChannels(ctx)
	assert.NoError(t, err)
	simdCw20Ics20Channel := simdChannels[len(simdChannels)-1]

	transferMsg := fmt.Sprintf(`{"channel": "%s", "remote_address": "%s"}`, simdCw20Ics20Channel.ChannelID, simdUser.FormattedAddress())
	transferMsgBase64 := base64.StdEncoding.EncodeToString([]byte(transferMsg))

	res, err := xplaChain.ExecuteContract(ctx, xplaUser.KeyName(), cw20ContractAddress, fmt.Sprintf(`{"send": {"contract": "%s", "amount": "%s", "msg": "%s"}}`, cw20Ics20ContractAddress, "1", transferMsgBase64), "--gas", "auto", "--gas-adjustment", "1.5")
	assert.NoError(t, err)
	assert.Equal(t, uint32(0), res.Code)

	err = ibcSetup.FlushRelayer(ctx)
	assert.NoError(t, err)

	simdBalances, err := ibcSetup.SimdChain.BankQueryAllBalances(ctx, simdUser.FormattedAddress())
	assert.NoError(t, err)
	assert.Equal(t, 2, len(simdBalances))

	for _, balance := range simdBalances {
		if balance.Denom != ibcSetup.SimdChain.Config().Denom {
			assert.Equal(t, sdkmath.NewInt(1), balance.Amount)
		}
	}

	err = xplaChain.QueryContract(ctx, cw20ContractAddress, fmt.Sprintf(`{
		"balance": {
			"address": "%s"
		}
	}`, xplaUser.FormattedAddress()), balanceResponse)

	assert.NoError(t, err)
	assert.Equal(t, "99999999", balanceResponse.Data.Balance)
}
