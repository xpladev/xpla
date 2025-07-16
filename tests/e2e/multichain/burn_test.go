package multichain_test

import (
	"context"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1types "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	interchaintest "github.com/cosmos/interchaintest/v10"
	"github.com/cosmos/interchaintest/v10/chain/cosmos"
	"github.com/cosmos/interchaintest/v10/ibc"
	"github.com/stretchr/testify/assert"
	"github.com/xpladev/xpla/tests/e2e/multichain"
	burntypes "github.com/xpladev/xpla/x/burn/types"
)

var (
	denom                = multichain.Denom
	defaultBurnAmount    = sdk.NewCoin(denom, sdkmath.NewInt(1_000_000_000_000_000_000))
	defaultDepositAmount = sdk.NewCoin(denom, sdkmath.NewInt(10_000_000))
	lessDepositAmount    = sdk.NewCoin(denom, sdkmath.NewInt(100_000))
	govAddress           string
	burnModuleAddress    string
	validatorKeyName     = "validator"
)

type testResult struct {
	proposal            *govv1types.Proposal
	status              govv1types.ProposalStatus
	diffSupplyBalance   sdkmath.Int
	diffProposerBalance sdkmath.Int
}

func TestMsgBurn(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	ctx := context.Background()
	chain, _ := multichain.StartXplaChain(t, ctx, multichain.LocalImage)

	xplaDefaultBalance, _ := sdkmath.NewIntFromString("10_000_000_000_000_000_000")
	xplaUsers := interchaintest.GetAndFundTestUsers(t, ctx, "default", xplaDefaultBalance, chain)
	user := xplaUsers[0]

	burnModuleAddress, _ = chain.AuthQueryModuleAddress(ctx, burntypes.ModuleName)
	govAddress, _ = chain.AuthQueryModuleAddress(ctx, govtypes.ModuleName)

	tests := []struct {
		title         string
		depositAmount sdk.Coin
		voteOpt       govv1types.VoteOption
		expected      testResult
	}{
		{
			"MsgBurn proposal, Vote YES",
			defaultDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_YES,
			testResult{
				status:              govv1types.ProposalStatus_PROPOSAL_STATUS_PASSED,
				diffSupplyBalance:   defaultBurnAmount.Amount,
				diffProposerBalance: defaultDepositAmount.Amount,
			},
		},
		{
			"MsgBurn proposal, Vote NO",
			defaultDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_NO,
			testResult{
				status:              govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
				diffSupplyBalance:   sdkmath.ZeroInt(),
				diffProposerBalance: defaultDepositAmount.Amount.Add(defaultBurnAmount.Amount),
			},
		},
		{
			"MsgBurn proposal, Vote ABSTAIN",
			defaultDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_ABSTAIN,
			testResult{
				status:              govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
				diffSupplyBalance:   sdkmath.ZeroInt(),
				diffProposerBalance: defaultDepositAmount.Amount.Add(defaultBurnAmount.Amount),
			},
		},
		{
			"MsgBurn proposal, Vote UNSPECIFIED",
			defaultDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_UNSPECIFIED,
			testResult{
				status:              govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
				diffSupplyBalance:   sdkmath.ZeroInt(),
				diffProposerBalance: defaultDepositAmount.Amount.Add(defaultBurnAmount.Amount),
			},
		},
		{
			"MsgBurn proposal, Vote NO_WITH_VETO",
			defaultDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_NO_WITH_VETO,
			testResult{
				status:              govv1types.ProposalStatus_PROPOSAL_STATUS_REJECTED,
				diffSupplyBalance:   defaultDepositAmount.Amount,
				diffProposerBalance: defaultBurnAmount.Amount,
			},
		},
		{
			"MsgBurn proposal, Vote UNSPECIFIED, less deposit",
			lessDepositAmount,
			govv1types.VoteOption_VOTE_OPTION_UNSPECIFIED,
			testResult{
				diffSupplyBalance:   sdkmath.ZeroInt(),
				diffProposerBalance: lessDepositAmount.Amount.Add(defaultBurnAmount.Amount),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.title, func(t *testing.T) {
			actual := testMsgBurnProposal(t, ctx, chain, user, tc.title, tc.depositAmount, tc.voteOpt)

			if tc.depositAmount.Equal(lessDepositAmount) {
				assert.Nil(t, actual.proposal)
			} else {
				assert.Equal(t, tc.expected.status, actual.proposal.Status)
			}

			assert.Equal(t, tc.expected.diffSupplyBalance.String(), actual.diffSupplyBalance.String())
			assert.Equal(t, tc.expected.diffProposerBalance.String(), actual.diffProposerBalance.String())
		})
	}
}

func testMsgBurnProposal(t *testing.T, ctx context.Context, chain *cosmos.CosmosChain, user ibc.Wallet, title string, depositAmount sdk.Coin, voteOpt govv1types.VoteOption) testResult {
	denom := chain.Config().Denom
	initialProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	initialBurnModuleBalance, _ := chain.GetBalance(ctx, burnModuleAddress, denom)
	initialGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)
	initialSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)

	t.Logf("Initial State - Proposer: %s, Gov Module: %s, Burn Module: %s, Supply: %s",
		initialProposerBalance.String(), initialGovBalance.String(), initialBurnModuleBalance.String(), initialSupply.String())

	msgBurn := &burntypes.MsgBurn{
		Authority: govAddress,
		Amount:    sdk.NewCoins(defaultBurnAmount),
	}

	proposalID, err := submitProposal(ctx, chain, user, []cosmos.ProtoMessage{msgBurn}, depositAmount, title, "Testing MsgBurn proposal")
	assert.NoError(t, err)

	processingProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	processingSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)
	processingBurnModuleBalance, _ := chain.GetBalance(ctx, burnModuleAddress, denom)
	processingGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)

	t.Logf("After Submission - Proposer: %s, Gov Module: %s, Burn Module: %s, Supply: %s",
		processingProposerBalance.String(), processingGovBalance.String(), processingBurnModuleBalance.String(), processingSupply.String())

	assert.True(t, processingProposerBalance.Add(defaultBurnAmount.Amount).Add(depositAmount.Amount).LT(initialProposerBalance))
	assert.Equal(t, initialSupply.String(), processingSupply.String())
	assert.Equal(t, initialBurnModuleBalance.Add(defaultBurnAmount.Amount).String(), processingBurnModuleBalance.String())
	assert.Equal(t, initialGovBalance.Add(depositAmount.Amount).String(), processingGovBalance.String())

	err = voteOnProposal(ctx, chain, validatorKeyName, proposalID, voteOpt)
	assert.NoError(t, err)

	err = multichain.WaitForBlocks(ctx, chain, 5)
	assert.NoError(t, err)

	prop, _ := chain.GovQueryProposalV1(ctx, proposalID)
	finalProposerBalance, _ := chain.GetBalance(ctx, user.FormattedAddress(), denom)
	finalSupply, _ := chain.BankQueryTotalSupplyOf(ctx, denom)
	finalBurnModuleBalance, _ := chain.GetBalance(ctx, burnModuleAddress, denom)
	finalGovBalance, _ := chain.GetBalance(ctx, govAddress, denom)

	t.Logf("Final State - Proposer: %s, Gov Module: %s, Burn Module: %s, Supply: %s",
		finalProposerBalance.String(), finalGovBalance.String(), finalBurnModuleBalance.String(), finalSupply.String())

	assert.Equal(t, sdkmath.ZeroInt(), finalBurnModuleBalance)
	assert.Equal(t, sdkmath.ZeroInt(), finalGovBalance)

	return testResult{
		proposal:            prop,
		diffSupplyBalance:   initialSupply.Amount.Sub(finalSupply.Amount),
		diffProposerBalance: finalProposerBalance.Sub(processingProposerBalance),
	}
}

func submitProposal(ctx context.Context, chain *cosmos.CosmosChain, proposer ibc.Wallet, messages []cosmos.ProtoMessage, depositAmount sdk.Coin, title, description string) (uint64, error) {
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
