package multichain_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	interchaintest "github.com/strangelove-ventures/interchaintest/v8"
	"github.com/strangelove-ventures/interchaintest/v8/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v8/ibc"
	"github.com/stretchr/testify/assert"
	"github.com/xpladev/xpla/tests/e2e/multichain"
	banktypes "github.com/xpladev/xpla/x/bank/types"
)

var (
	denom            = multichain.Denom
	burnAmount       = sdk.NewCoin(denom, sdkmath.NewInt(1_000_000_000_000_000_000))
	depositAmount    = sdk.NewCoin(denom, sdkmath.NewInt(10_000_000))
	govAddress       string
	validatorKeyName = "validator"
)

func TestMsgBurn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	chain, _ := multichain.StartXplaChain(t, ctx, multichain.LocalImage)

	xplaDefaultBalance, _ := sdkmath.NewIntFromString("10_000_000_000_000_000_000")
	xplaUsers := interchaintest.GetAndFundTestUsers(t, ctx, "default", xplaDefaultBalance, chain)
	user := xplaUsers[0]

	govAddress, _ = chain.AuthQueryModuleAddress(ctx, govtypes.ModuleName)

	t.Run("Proposal Success", func(t *testing.T) {
		testMsgBurnProposal(t, ctx, chain, user, govv1types.VoteOption_VOTE_OPTION_YES)
	})
	t.Run("Proposal Rejection", func(t *testing.T) {
		testMsgBurnProposal(t, ctx, chain, user, govv1types.VoteOption_VOTE_OPTION_NO)
	})
	t.Run("Proposal Veto", func(t *testing.T) {
		testMsgBurnProposal(t, ctx, chain, user, govv1types.VoteOption_VOTE_OPTION_NO_WITH_VETO)
	})
}

func testMsgBurnProposal(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, voteOpt govv1types.VoteOption) {
	denom := chain.Config().Denom
	initialProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	initialGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)
	initialSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)

	t.Logf("Initial State - Proposer: %s, Gov: %s, Supply: %s",
		initialProposerBalance.String(), initialGovBalance.String(), initialSupply.String())

	msgBurn := &banktypes.MsgBurn{
		Authority: govAddress,
		Amount:    sdk.NewCoins(burnAmount),
	}

	proposalID, err := submitProposal(ctx, chain, user, []cosmos.ProtoMessage{msgBurn}, "Test MsgBurn "+voteOpt.String(), "Testing MsgBurn proposal")
	assert.NoError(t, err)

	processingProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	processingSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)
	processingGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)

	t.Logf("After Submission - Proposer: %s, Gov: %s, Supply: %s",
		processingProposerBalance.String(), processingGovBalance.String(), processingSupply.String())

	assert.True(t, processingProposerBalance.Add(burnAmount.Amount).Add(depositAmount.Amount).LT(initialProposerBalance))
	assert.Equal(t, initialSupply.String(), processingSupply.String())
	assert.Equal(t, initialGovBalance.Add(depositAmount.Amount).Add(burnAmount.Amount).String(), processingGovBalance.String())

	err = voteOnProposal(ctx, chain, validatorKeyName, proposalID, voteOpt)
	assert.NoError(t, err)

	err = multichain.WaitForBlocks(ctx, chain, 5)
	assert.NoError(t, err)

	prop, _ := chain.GovQueryProposalV1(ctx, proposalID)
	finalProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	finalSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)
	finalGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)

	t.Logf("Final State - Proposer: %s, Gov: %s, Supply: %s, Proposal Status: %s",
		finalProposerBalance.String(), finalGovBalance.String(), finalSupply.String(), prop.Status.String())

	switch voteOpt {
	case govv1types.VoteOption_VOTE_OPTION_YES:
		assert.Equal(t, govv1types.ProposalStatus_PROPOSAL_STATUS_PASSED, prop.Status)
		assert.Equal(t, initialSupply.Sub(burnAmount).String(), finalSupply.String())
		assert.Equal(t, processingProposerBalance.Add(depositAmount.Amount).String(), finalProposerBalance.String())
		assert.Equal(t, processingGovBalance.Sub(depositAmount.Amount).Sub(burnAmount.Amount).String(), finalGovBalance.String())

	case govv1types.VoteOption_VOTE_OPTION_NO:
		assert.Equal(t, govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED, prop.Status)
		assert.Equal(t, initialSupply.String(), finalSupply.String())
		assert.Equal(t, processingProposerBalance.Add(depositAmount.Amount).Add(burnAmount.Amount).String(), finalProposerBalance.String())
		assert.Equal(t, processingGovBalance.Sub(depositAmount.Amount).Sub(burnAmount.Amount).String(), finalGovBalance.String())

	case govv1types.VoteOption_VOTE_OPTION_NO_WITH_VETO:
		assert.Equal(t, govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED, prop.Status)
		assert.Equal(t, initialSupply.Sub(depositAmount).String(), finalSupply.String())
		assert.Equal(t, processingProposerBalance.Add(burnAmount.Amount).String(), finalProposerBalance.String())
		assert.Equal(t, processingGovBalance.Sub(depositAmount.Amount).Sub(burnAmount.Amount).String(), finalGovBalance.String())
	}
}

func submitProposal(ctx context.Context, chain *cosmos.CosmosChain, proposer ibc.Wallet, messages []cosmos.ProtoMessage, title, description string) (uint64, error) {
	proposal, err := chain.BuildProposal(
		messages,
		title,
		description,
		"",
		depositAmount.String(),
		proposer.FormattedAddress(),
		false,
	)
	if err != nil {
		return 0, err
	}
	_, err = chain.GetNode().SubmitProposal(ctx, proposer.KeyName(), proposal)
	if err != nil {
		return 0, err
	}

	// Get latest proposal id
	proposals, err := chain.GovQueryProposalsV1(ctx, 0)
	if err != nil || len(proposals) == 0 {
		return 0, err
	}

	return proposals[len(proposals)-1].Id, nil
}

func voteOnProposal(ctx context.Context, chain *cosmos.CosmosChain, validatorKeyName string, proposalID uint64, option govv1types.VoteOption) error {
	var voteStr string
	switch option {
	case govv1types.VoteOption_VOTE_OPTION_YES:
		voteStr = "yes"
	case govv1types.VoteOption_VOTE_OPTION_NO:
		voteStr = "no"
	case govv1types.VoteOption_VOTE_OPTION_NO_WITH_VETO:
		voteStr = "no_with_veto"
	case govv1types.VoteOption_VOTE_OPTION_ABSTAIN:
		voteStr = "abstain"
	default:
		return nil
	}
	return chain.GetNode().VoteOnProposal(ctx, validatorKeyName, proposalID, voteStr)
}
