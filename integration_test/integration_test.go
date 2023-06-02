package integrationtest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	wasmtype "github.com/CosmWasm/wasmd/x/wasm/types"
	ed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	govtype "github.com/cosmos/cosmos-sdk/x/gov/types"
	paramtype "github.com/cosmos/cosmos-sdk/x/params/types/proposal"
	slashingtype "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtype "github.com/cosmos/cosmos-sdk/x/staking/types"

	xplatypes "github.com/xpladev/xpla/types"
	volunteerValType "github.com/xpladev/xpla/x/volunteer/types"

	// evmtypes "github.com/evmos/ethermint/x/evm/types"

	abibind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	web3 "github.com/ethereum/go-ethereum/ethclient"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

func TestWasmContractAndTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	testSuite := &WASMIntegrationTestSuite{}
	suite.Run(t, testSuite)
}

type WASMIntegrationTestSuite struct {
	suite.Suite

	TokenAddress string

	UserWallet1               *WalletInfo
	UserWallet2               *WalletInfo
	ValidatorWallet1          *WalletInfo
	ValidatorWallet2          *WalletInfo
	ValidatorWallet3          *WalletInfo
	ValidatorWallet4          *WalletInfo
	ValidatorWallet5          *WalletInfo
	VolunteerValidatorWallet1 *WalletInfo
	VolunteerValidatorWallet2 *WalletInfo
	VolunteerValidatorWallet3 *WalletInfo

	Validator1PVKey          *PVKey
	VolunteerValidatorPVKey1 *PVKey
	VolunteerValidatorPVKey2 *PVKey
	VolunteerValidatorPVKey3 *PVKey
	Validator5PVKey          *PVKey
}

func (i *WASMIntegrationTestSuite) SetupSuite() {
	desc = NewServiceDesc("127.0.0.1", 19090, 10, true)

	i.UserWallet1,
		i.UserWallet2,
		i.ValidatorWallet1,
		i.ValidatorWallet2,
		i.ValidatorWallet3,
		i.ValidatorWallet4,
		i.ValidatorWallet5,
		i.VolunteerValidatorWallet1,
		i.VolunteerValidatorWallet2,
		i.VolunteerValidatorWallet3 = walletSetup()
}

func (i *WASMIntegrationTestSuite) SetupTest() {
	i.UserWallet1,
		i.UserWallet2,
		i.ValidatorWallet1,
		i.ValidatorWallet2,
		i.ValidatorWallet3,
		i.ValidatorWallet4,
		i.ValidatorWallet5,
		i.VolunteerValidatorWallet1,
		i.VolunteerValidatorWallet2,
		i.VolunteerValidatorWallet3 = walletSetup()

	i.UserWallet1.RefreshSequence()
	i.UserWallet2.RefreshSequence()
	i.ValidatorWallet1.RefreshSequence()
	i.ValidatorWallet2.RefreshSequence()
	i.ValidatorWallet3.RefreshSequence()
	i.ValidatorWallet4.RefreshSequence()
	i.ValidatorWallet5.RefreshSequence()
	i.VolunteerValidatorWallet1.RefreshSequence()

	var err error
	i.VolunteerValidatorPVKey1, err = loadPrivValidator("volunteer_validator1")
	if err != nil {
		i.Fail("PVKey load fail - 1")
	}

	i.VolunteerValidatorPVKey2, err = loadPrivValidator("volunteer_validator2")
	if err != nil {
		i.Fail("PVKey load fail - 2")
	}

	i.VolunteerValidatorPVKey3, err = loadPrivValidator("volunteer_validator3")
	if err != nil {
		i.Fail("PVKey load fail - 3")
	}

	i.Validator1PVKey, err = loadPrivValidator("validator1")
	if err != nil {
		i.Fail("PVKey load fail - validator 1")
	}

	i.Validator5PVKey, err = loadPrivValidator("validator5_experimental")
	if err != nil {
		i.Fail("PVKey load fail")
	}
}

func (i *WASMIntegrationTestSuite) TearDownTest() {}

func (u *WASMIntegrationTestSuite) TearDownSuite() {
	desc.CloseConnection()
}

// Test strategy
// 1. Simple delegation
// 2. Contract upload
// 3. Contract initiate
// 4. Contract execute

func (t *WASMIntegrationTestSuite) Test01_SimpleDelegation() {
	amt := sdktypes.NewInt(100000000000000)
	coin := sdktypes.NewCoin(xplatypes.DefaultDenom, amt)

	delegationMsg := stakingtype.NewMsgDelegate(
		t.UserWallet1.ByteAddress,
		t.ValidatorWallet1.ByteAddress.Bytes(),
		coin,
	)

	feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

	txhash, err := t.UserWallet1.SendTx(ChainID, delegationMsg, fee, xplaGeneralGasLimit, false)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), txhash)

	err = txCheck(txhash)
	assert.NoError(t.T(), err)

	queryClient := stakingtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryDelegatorMsg := &stakingtype.QueryDelegatorDelegationsRequest{
		DelegatorAddr: t.UserWallet1.StringAddress,
	}

	delegationResp, err := queryClient.DelegatorDelegations(context.Background(), queryDelegatorMsg)
	assert.NoError(t.T(), err)

	delegatedList := delegationResp.GetDelegationResponses()

	expected := []stakingtype.DelegationResponse{
		stakingtype.NewDelegationResp(
			t.UserWallet1.ByteAddress,
			t.ValidatorWallet1.ByteAddress.Bytes(),
			sdktypes.NewDecFromInt(amt),
			sdktypes.NewCoin(xplatypes.DefaultDenom, amt),
		),
	}

	assert.Equal(t.T(), expected, delegatedList)
}

func (t *WASMIntegrationTestSuite) Test02_StoreCode() {
	contractBytes, err := os.ReadFile(filepath.Join(".", "misc", "token.wasm"))
	if err != nil {
		panic(err)
	}

	storeMsg := &wasmtype.MsgStoreCode{
		Sender:       t.UserWallet1.StringAddress,
		WASMByteCode: contractBytes,
	}

	feeAmt := sdktypes.NewDec(xplaCodeGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())
	txhash, err := t.UserWallet1.SendTx(ChainID, storeMsg, fee, xplaCodeGasLimit, false)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), txhash)

	err = txCheck(txhash)
	assert.NoError(t.T(), err)

	queryClient := wasmtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryCodeMsg := &wasmtype.QueryCodeRequest{
		CodeId: 1,
	}

	resp, err := queryClient.Code(context.Background(), queryCodeMsg)

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), resp)
}

func (t *WASMIntegrationTestSuite) Test03_InstantiateContract() {
	initMsg := []byte(fmt.Sprintf(`
		{
			"name": "testtoken",
			"symbol": "TKN",
			"decimals": 6,
			"initial_balances": [
				{
					"address": "%s",
					"amount": "100000000"
				}
			]
		}
	`, t.UserWallet2.StringAddress))

	instantiateMsg := &wasmtype.MsgInstantiateContract{
		Sender: t.UserWallet2.StringAddress,
		Admin:  t.UserWallet2.StringAddress,
		CodeID: 1,
		Label:  "Integration test purpose",
		Msg:    initMsg,
	}

	feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

	txhash, err := t.UserWallet2.SendTx(ChainID, instantiateMsg, fee, xplaGeneralGasLimit, false)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), txhash)

	err = txCheck(txhash)
	assert.NoError(t.T(), err)

	queryClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
	resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
		Hash: txhash,
	})

	assert.NoError(t.T(), err)

ATTR:
	for _, val := range resp.TxResponse.Events {
		for _, attr := range val.Attributes {
			if string(attr.Key) == "_contract_address" {
				t.TokenAddress = string(attr.Value)
				break ATTR
			}
		}
	}

	queryTokenAmtClient := wasmtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryStr := []byte(fmt.Sprintf(`{
		"balance": {
			"address": "%s"
		}
	}`, t.UserWallet2.StringAddress))

	tokenResp, err := queryTokenAmtClient.SmartContractState(context.Background(), &wasmtype.QuerySmartContractStateRequest{
		Address:   t.TokenAddress,
		QueryData: queryStr,
	})

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenResp)

	type AmtResp struct {
		Balance string `json:"balance"`
	}

	amtResp := &AmtResp{}
	err = json.Unmarshal(tokenResp.Data.Bytes(), amtResp)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), "100000000", amtResp.Balance)
}

func (t *WASMIntegrationTestSuite) Test04_ContractExecution() {
	transferMsg := []byte(fmt.Sprintf(`
		{
			"transfer": {
				"recipient": "%s",
				"amount": "50000000"
			}
		}
	`, t.UserWallet1.StringAddress))

	contractExecMsg := &wasmtype.MsgExecuteContract{
		Sender:   t.UserWallet2.StringAddress,
		Contract: t.TokenAddress,
		Msg:      transferMsg,
	}

	feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

	txhash, err := t.UserWallet2.SendTx(ChainID, contractExecMsg, fee, xplaGeneralGasLimit, false)
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), txhash)

	err = txCheck(txhash)
	assert.NoError(t.T(), err)

	queryTokenAmtClient := wasmtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryStr := []byte(fmt.Sprintf(`{
		"balance": {
			"address": "%s"
		}
	}`, t.UserWallet2.StringAddress))

	tokenResp, err := queryTokenAmtClient.SmartContractState(context.Background(), &wasmtype.QuerySmartContractStateRequest{
		Address:   t.TokenAddress,
		QueryData: queryStr,
	})

	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), tokenResp)

	type AmtResp struct {
		Balance string `json:"balance"`
	}

	amtResp := &AmtResp{}
	err = json.Unmarshal(tokenResp.Data.Bytes(), amtResp)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), "50000000", amtResp.Balance)
}

func (t *WASMIntegrationTestSuite) Test12_GeneralVolunteerValidatorRegistryUnregistryDelegation() {
	amt := sdktypes.NewInt(1000000000000000000)

	{
		fmt.Println("Preparing proposal to add a volunteer validator")

		proposalContent, err := volunteerValType.NewRegisterVolunteerValidatorProposal(
			"register_volunteer_validator",
			"Test volunteer validator registary",
			sdktypes.AccAddress(t.VolunteerValidatorWallet1.ByteAddress.Bytes()),
			sdktypes.ValAddress(t.VolunteerValidatorWallet1.ByteAddress.Bytes()),
			&ed25519.PubKey{Key: t.VolunteerValidatorPVKey1.PubKey.Bytes()},
			sdktypes.NewCoin(xplatypes.DefaultDenom, amt), // smaller amount than other basic validators
			stakingtype.NewDescription("volunteer_validator_1", "", "", "", ""),
		)

		assert.NoError(t.T(), err)

		err = applyVoteTallyingProposal(
			desc.GetConnectionWithContext(context.Background()),
			proposalContent,
			t.VolunteerValidatorWallet1,
			[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
		)

		assert.NoError(t.T(), err)
	}

	fmt.Println("Waiting for validator committing...")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		fmt.Println("Validator status check")

		didVolunteerValVote, err := checkValidatorVoted(
			desc.GetServiceDesc().ServiceConn,
			t.VolunteerValidatorPVKey1.Address,
		)
		assert.NoError(t.T(), err)

		if assert.True(t.T(), didVolunteerValVote) {
			fmt.Println("Volunteer validator voted. Succeeded")
		} else {
			fmt.Println("Volunteer validator did not vote. Test fail")
		}
	}

	{
		fmt.Println("Delegator status check")

		queryClient := stakingtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

		queryDelegatorMsg := &stakingtype.QueryDelegatorDelegationsRequest{
			DelegatorAddr: t.VolunteerValidatorWallet1.StringAddress,
		}

		delegationResp, err := queryClient.DelegatorDelegations(context.Background(), queryDelegatorMsg)
		assert.NoError(t.T(), err)

		delegatedList := delegationResp.GetDelegationResponses()

		expected := []stakingtype.DelegationResponse{
			stakingtype.NewDelegationResp(
				t.VolunteerValidatorWallet1.ByteAddress,
				t.VolunteerValidatorWallet1.ByteAddress.Bytes(),
				sdktypes.NewDecFromInt(amt),
				sdktypes.NewCoin(xplatypes.DefaultDenom, amt),
			),
		}

		if assert.Equal(t.T(), expected, delegatedList) {
			fmt.Println("Only one delegator exists. Check OK")
		} else {
			fmt.Println("Something wrong in the module")
			t.T().Fail()
		}
	}

	delegationAmt := sdktypes.NewInt(100000000000000)
	{
		fmt.Println("Try delegation and should fail...")

		coin := sdktypes.NewCoin(xplatypes.DefaultDenom, delegationAmt)

		delegationMsg := stakingtype.NewMsgDelegate(
			t.UserWallet1.ByteAddress,
			t.VolunteerValidatorWallet1.ByteAddress.Bytes(),
			coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.UserWallet1.SendTx(ChainID, delegationMsg, fee, xplaGeneralGasLimit, false)
		if assert.Error(t.T(), err) && assert.Equal(t.T(), txhash, "") {
			fmt.Println("Expected failure is occurred.")
		} else {
			fmt.Println("Tx sent. Test fail")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Try redelegation and should fail...")

		coin := sdktypes.NewCoin(xplatypes.DefaultDenom, delegationAmt)

		redelegationMsg := stakingtype.NewMsgBeginRedelegate(
			t.UserWallet1.ByteAddress,
			t.ValidatorWallet1.ByteAddress.Bytes(),
			t.VolunteerValidatorWallet1.ByteAddress.Bytes(),
			coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.UserWallet1.SendTx(ChainID, redelegationMsg, fee, xplaGeneralGasLimit, false)
		if assert.Error(t.T(), err) && assert.Equal(t.T(), txhash, "") {
			fmt.Println("Expected failure is occurred.")
		} else {
			fmt.Println("Tx sent. Test fail")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Try undelegation and should fail...")

		coin := sdktypes.NewCoin(xplatypes.DefaultDenom, delegationAmt)

		redelegationMsg := stakingtype.NewMsgUndelegate(
			t.VolunteerValidatorWallet1.ByteAddress.Bytes(),
			t.VolunteerValidatorWallet1.ByteAddress.Bytes(),
			coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.VolunteerValidatorWallet1.SendTx(ChainID, redelegationMsg, fee, xplaGeneralGasLimit, false)
		if assert.Error(t.T(), err) && assert.Equal(t.T(), txhash, "") {
			fmt.Println("Expected failure is occurred.")
		} else {
			fmt.Println("Tx sent. Test fail")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Preparing proposal to remove a volunteer validator")

		proposalContent := volunteerValType.NewUnregisterVolunteerValidatorProposal(
			"unregister_volunteer_validator",
			"Test volunteer validator unregistration",
			sdktypes.ValAddress(t.VolunteerValidatorWallet1.ByteAddress.Bytes()),
		)

		err := applyVoteTallyingProposal(
			desc.GetConnectionWithContext(context.Background()),
			proposalContent,
			t.VolunteerValidatorWallet1,
			[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
		)

		assert.NoError(t.T(), err)
	}

	fmt.Println("Waiting some blocks for the proposal passing...")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		fmt.Println("Check existence of the volunteer validator")

		client := volunteerValType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.VolunteerValidators(context.Background(), &volunteerValType.QueryVolunteerValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVolunteerValAddress := sdktypes.ValAddress(t.VolunteerValidatorWallet1.ByteAddress).String()

		if assert.NotContains(t.T(), validatorStatus.GetVolunteerValidators(), thisVolunteerValAddress) {
			fmt.Println(thisVolunteerValAddress, "is successfully removed from validator set!")
		} else {
			fmt.Println(thisVolunteerValAddress, "still found")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Try deregister a validator but it is not registered...")

		proposalContent := volunteerValType.NewUnregisterVolunteerValidatorProposal(
			"false_unregister_volunteer_validator",
			"False volunteer validator unregistration",
			sdktypes.ValAddress(t.UserWallet1.ByteAddress.Bytes()),
		)

		proposalMsg, err := govtype.NewMsgSubmitProposal(
			proposalContent,
			sdktypes.NewCoins(sdktypes.NewCoin(xplatypes.DefaultDenom, sdktypes.NewInt(10000000))),
			t.UserWallet1.ByteAddress,
		)

		assert.NoError(t.T(), err)

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.UserWallet1.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
		if assert.Equal(t.T(), "", txhash) && assert.Error(t.T(), err) {
			fmt.Println(err)
			fmt.Println("Expected failure! Test succeeded!")
		} else {
			fmt.Println("Tx sent as", txhash, "Unexpected situation")
			t.T().Fail()
		}
	}
}

func (t *WASMIntegrationTestSuite) Test13_MultipleProposals() {
	amt := sdktypes.NewInt(1000000000000000000)

	{
		fmt.Println("Preparing multiple proposals to add a volunteer validator")

		var eg errgroup.Group

		for i := 0; i < 2; i++ {
			eg.Go(func() error {
				// apply proposal
				proposalContent, err := volunteerValType.NewRegisterVolunteerValidatorProposal(
					fmt.Sprintf("register_multiple_volunteer_validator_%d", i),
					fmt.Sprintf("Test volunteer validator registary_%d", i),
					sdktypes.AccAddress(t.VolunteerValidatorWallet2.ByteAddress.Bytes()),
					sdktypes.ValAddress(t.VolunteerValidatorWallet2.ByteAddress.Bytes()),
					&ed25519.PubKey{Key: t.VolunteerValidatorPVKey2.PubKey.Bytes()},
					sdktypes.NewCoin(xplatypes.DefaultDenom, amt), // smaller amount than other basic validators
					stakingtype.NewDescription("volunteer_validator_2", "", "", "", ""),
				)

				if err != nil {
					return err
				}

				err = applyVoteTallyingProposal(
					desc.GetConnectionWithContext(context.Background()),
					proposalContent,
					t.VolunteerValidatorWallet2,
					[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
				)

				if err != nil {
					return err
				}

				return nil
			})
		}

		err := eg.Wait()

		if assert.NoError(t.T(), err) {
			fmt.Println("Proposal successfully applied!")
		} else {
			fmt.Println("Error detected on the proposal")
			t.T().Fail()
		}
	}

	fmt.Println("Waiting some blocks for the proposal passing...")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		fmt.Println("Check existence of the volunteer validator")

		client := volunteerValType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.VolunteerValidators(context.Background(), &volunteerValType.QueryVolunteerValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVolunteerValAddress := sdktypes.ValAddress(t.VolunteerValidatorWallet2.ByteAddress).String()

		if len(validatorStatus.GetVolunteerValidators()) == 1 &&
			assert.Contains(t.T(), validatorStatus.GetVolunteerValidators(), thisVolunteerValAddress) {
			fmt.Println(thisVolunteerValAddress, "successfully get in the validator set!")
		} else {
			fmt.Println(thisVolunteerValAddress, "does not exist")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Preparing multiple proposals to remove a volunteer validator")

		var eg errgroup.Group

		for i := 0; i < 2; i++ {
			eg.Go(func() error {
				// apply proposal
				proposalContent := volunteerValType.NewUnregisterVolunteerValidatorProposal(
					fmt.Sprintf("unregister_multiple_volunteer_validator_%d", i),
					fmt.Sprintf("Test volunteer validator unregistary_%d", i),
					sdktypes.ValAddress(t.VolunteerValidatorWallet2.ByteAddress.Bytes()),
				)

				err := applyVoteTallyingProposal(
					desc.GetConnectionWithContext(context.Background()),
					proposalContent,
					t.VolunteerValidatorWallet2,
					[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
				)

				if err != nil {
					return err
				}

				return nil
			})

			// slight delay for avoiding sequence mismatch
			time.Sleep(time.Second / 5)
		}

		err := eg.Wait()

		if assert.NoError(t.T(), err) {
			fmt.Println("Proposal successfully applied!")
		} else {
			fmt.Println("Error detected on the proposal")
			t.T().Fail()
		}
	}

	fmt.Println("Waiting some blocks for the proposal passing...")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		fmt.Println("Check existence of the volunteer validator")

		client := volunteerValType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.VolunteerValidators(context.Background(), &volunteerValType.QueryVolunteerValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVolunteerValAddress := sdktypes.ValAddress(t.VolunteerValidatorWallet2.ByteAddress).String()

		if assert.NotContains(t.T(), validatorStatus.GetVolunteerValidators(), thisVolunteerValAddress) {
			fmt.Println(thisVolunteerValAddress, "successfully removed from the validator set!")
		} else {
			fmt.Println(thisVolunteerValAddress, "still exist!")
			t.T().Fail()
		}
	}
}

func (t *WASMIntegrationTestSuite) Test14_TryChangingGeneralValidatorToVolunteerValidator_ShouldFail() {
	amt := sdktypes.NewInt(1_000000_000000_000000)

	{
		fmt.Println("Try registering as a volunteer validator from the general validator...")

		proposalContent, err := volunteerValType.NewRegisterVolunteerValidatorProposal(
			"register_volunteer_validator_from_general_validator",
			"Test volunteer validator registary",
			sdktypes.AccAddress(t.ValidatorWallet1.ByteAddress.Bytes()),
			sdktypes.ValAddress(t.ValidatorWallet1.ByteAddress.Bytes()),
			&ed25519.PubKey{Key: t.Validator1PVKey.PubKey.Bytes()},
			sdktypes.NewCoin(xplatypes.DefaultDenom, amt), // smaller amount than other basic validators
			stakingtype.NewDescription("volunteer_validator", "", "", "", ""),
		)

		assert.NoError(t.T(), err)

		err = applyVoteTallyingProposal(
			desc.GetConnectionWithContext(context.Background()),
			proposalContent,
			t.VolunteerValidatorWallet2,
			[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
		)

		if assert.Error(t.T(), err) {
			fmt.Println(err)
			fmt.Println("Expected failure! Test succeeded!")
		} else {
			fmt.Println("Unexpected situation. Failure")
			t.T().Fail()
		}
	}
}

func (t *WASMIntegrationTestSuite) Test15_ValidatorActiveSetChange() {
	amt := sdktypes.NewInt(1_200000_000000_000000)
	var maxValidators uint32 = 5

	expectedChange := []paramtype.ParamChange{
		paramtype.NewParamChange(stakingtype.ModuleName, string(stakingtype.KeyMaxValidators), fmt.Sprintf("%d", maxValidators)),
		paramtype.NewParamChange(slashingtype.ModuleName, string(slashingtype.KeySignedBlocksWindow), "\"5\""),
		paramtype.NewParamChange(slashingtype.ModuleName, string(slashingtype.KeyDowntimeJailDuration), "\"10\""),
	}

	{
		fmt.Println("Decrease the number of active set")
		fmt.Println("Current # of validator:", maxValidators)

		proposalContent := paramtype.NewParameterChangeProposal(
			"decrease_validator_active_set",
			"Decrease validator active set",
			expectedChange,
		)

		err := applyVoteTallyingProposal(
			desc.GetConnectionWithContext(context.Background()),
			proposalContent,
			t.ValidatorWallet2,
			[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
		)

		assert.NoError(t.T(), err)
	}

	{
		fmt.Println("Add one more volunteer validator")
		proposalContent, err := volunteerValType.NewRegisterVolunteerValidatorProposal(
			"register_volunteer_validator_3",
			"Test volunteer validator registry3 ",
			sdktypes.AccAddress(t.VolunteerValidatorWallet3.ByteAddress.Bytes()),
			sdktypes.ValAddress(t.VolunteerValidatorWallet3.ByteAddress.Bytes()),
			&ed25519.PubKey{Key: t.VolunteerValidatorPVKey3.PubKey.Bytes()},
			sdktypes.NewCoin(xplatypes.DefaultDenom, amt), // smaller amount than other basic validators
			stakingtype.NewDescription("volunteer_validator_3", "", "", "", ""),
		)

		assert.NoError(t.T(), err)

		err = applyVoteTallyingProposal(
			desc.GetConnectionWithContext(context.Background()),
			proposalContent,
			t.VolunteerValidatorWallet3,
			[]*WalletInfo{t.ValidatorWallet1, t.ValidatorWallet2, t.ValidatorWallet3, t.ValidatorWallet4},
		)

		assert.NoError(t.T(), err)
	}

	{
		fmt.Println("Add normal validator")

		// more than volunteer validator but less than other validator
		delegationAmt := sdktypes.NewCoin(xplatypes.DefaultDenom, sdktypes.NewInt(1_500000_000000_000000))

		createValidatorMsg, err := stakingtype.NewMsgCreateValidator(
			sdktypes.ValAddress(t.ValidatorWallet5.ByteAddress.Bytes()),
			&ed25519.PubKey{Key: t.Validator5PVKey.PubKey.Bytes()},
			delegationAmt,
			stakingtype.NewDescription("validator5", "", "", "", ""),
			stakingtype.NewCommissionRates(
				sdktypes.MustNewDecFromStr("0.1"),
				sdktypes.MustNewDecFromStr("0.2"),
				sdktypes.MustNewDecFromStr("0.01"),
			),
			sdktypes.NewInt(1),
		)

		assert.NoError(t.T(), err)

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.ValidatorWallet5.SendTx(ChainID, createValidatorMsg, fee, xplaProposalGasLimit, false)
		if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
			fmt.Println("Tx sent", txhash)
		} else {
			fmt.Println(err)
		}

		err = txCheck(txhash)
		if assert.NoError(t.T(), err) {
			fmt.Println("Tx applied", txhash)
		} else {
			fmt.Println(err)
		}
	}

	fmt.Println("Waiting some blocks for the validator status refresh...")
	fmt.Println("Expected situation: 5 normal validators + 1 volunteer validator = 6 validators")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		valList, err := getValidatorListOfLatestBlock(desc.GetServiceDesc().ServiceConn)
		assert.NoError(t.T(), err)

		if assert.Equal(t.T(), int(maxValidators+1), len(valList)) {
			fmt.Println("Matched expectation!")
		} else {
			fmt.Println("Not matched expectation. Test fail! Expected:", maxValidators, "Actual:", len(valList))
		}

		fmt.Println("Check the volunteer validator voted...")

		found := false
		for _, unitVal := range valList {
			fmt.Println(unitVal.String())
			if t.VolunteerValidatorPVKey3.Address.String() == unitVal.String() {
				found = true
			}
		}

		if assert.True(t.T(), found) {
			fmt.Println("Volunteer validator voted. Succeeded")
		} else {
			fmt.Println("Volunteer validator did not vote. Test fail")
		}

		// validator 5 bonded?
	}

	{
		fmt.Println("Add one more normal validator. But its power is smaller than the volunteer validator.")
		fmt.Println("Expected unbonded state")

		// // more than volunteer validator but less than other validator
		// delegationAmt := sdktypes.NewCoin(xplatypes.DefaultDenom, sdktypes.NewInt(1_500000_000000_000000))

		// createValidatorMsg, err := stakingtype.NewMsgCreateValidator(
		// 	sdktypes.ValAddress(t.ValidatorWallet5.ByteAddress.Bytes()),
		// 	&ed25519.PubKey{Key: t.Validator5PVKey.PubKey.Bytes()},
		// 	delegationAmt,
		// 	stakingtype.NewDescription("validator5", "", "", "", ""),
		// 	stakingtype.NewCommissionRates(
		// 		sdktypes.MustNewDecFromStr("0.1"),
		// 		sdktypes.MustNewDecFromStr("0.2"),
		// 		sdktypes.MustNewDecFromStr("0.01"),
		// 	),
		// 	sdktypes.NewInt(1),
		// )

		// assert.NoError(t.T(), err)

		// feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		// fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		// txhash, err := t.ValidatorWallet5.SendTx(ChainID, createValidatorMsg, fee, xplaProposalGasLimit, false)
		// if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
		// 	fmt.Println("Tx sent", txhash)
		// } else {
		// 	fmt.Println(err)
		// }

		// err = txCheck(txhash)
		// if assert.NoError(t.T(), err) {
		// 	fmt.Println("Tx applied", txhash)
		// } else {
		// 	fmt.Println(err)
		// }
	}

	{
		fmt.Println("Try to turn off the volunteer validator")

		cmd := exec.Command("docker", "stop", "xpla-localnet-volunteer3")
		err := cmd.Run()
		assert.NoError(t.T(), err)

		fmt.Println("Volunteer validator down. Wait 5 blocktime to be jailed")
		time.Sleep(time.Second*blocktime*jailBlocks + 1)
		fmt.Println("Wait for validator list reorg..")
		time.Sleep(time.Second*blocktime*4 + 1)

		didVolunteerValVote, err := checkValidatorVoted(
			desc.GetServiceDesc().ServiceConn,
			t.VolunteerValidatorPVKey3.Address,
		)
		assert.NoError(t.T(), err)

		if assert.False(t.T(), didVolunteerValVote) {
			fmt.Println("Volunteer validator did not vote. Succeeded")
		} else {
			fmt.Println("Volunteer validator voted. Test fail")
		}
	}

	{
		fmt.Println("Turn on the volunteer validator and unjailing")

		cmd := exec.Command("docker", "start", "xpla-localnet-volunteer3")
		err := cmd.Run()
		assert.NoError(t.T(), err)

		fmt.Println("Wait enough time(30sec) to replay the blocks...")
		time.Sleep(time.Second * 30)

		unjailMsg := slashingtype.NewMsgUnjail(sdktypes.ValAddress(t.VolunteerValidatorWallet3.ByteAddress))

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := t.VolunteerValidatorWallet3.SendTx(ChainID, unjailMsg, fee, xplaGeneralGasLimit, false)
		if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
			fmt.Println("Tx sent", txhash)
		} else {
			fmt.Println(err)
		}

		err = txCheck(txhash)
		if assert.NoError(t.T(), err) {
			fmt.Println("Tx applied", txhash)
		} else {
			fmt.Println(err)
		}

	}

	fmt.Println("Waiting some block time for the validator status refresh...")
	time.Sleep(time.Second*blocktime*validatorActiveBlocks + 1)

	{
		valList, err := getValidatorListOfLatestBlock(desc.GetServiceDesc().ServiceConn)
		assert.NoError(t.T(), err)

		if assert.Equal(t.T(), int(maxValidators+1), len(valList)) {
			fmt.Println("Matched expectation!")
		} else {
			fmt.Println("Not matched expectation. Test fail! Expected:", maxValidators, "Actual:", len(valList))
		}

		fmt.Println("Check the volunteer validator voted...")

		found := false
		for _, unitVal := range valList {
			fmt.Println(unitVal.String())
			if t.VolunteerValidatorPVKey3.Address.String() == unitVal.String() {
				found = true
			}
		}

		if assert.True(t.T(), found) {
			fmt.Println("Volunteer validator voted. Succeeded")
		} else {
			fmt.Println("Volunteer validator did not vote. Test fail")
		}
	}
}

// Test strategy
// 1. Check balance
// 2. Contract upload & initiate
// 3. Contract execute
// 4. Contract execute by Cosmos SDK Tx -> available from ethermint v0.20

func TestEVMContractAndTx(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	testSuite := &EVMIntegrationTestSuite{}
	suite.Run(t, testSuite)
}

type EVMIntegrationTestSuite struct {
	suite.Suite

	EthClient *web3.Client

	Coinbase     string
	TokenAddress ethcommon.Address

	UserWallet1      *EVMWalletInfo
	UserWallet2      *EVMWalletInfo
	ValidatorWallet1 *EVMWalletInfo
}

func (t *EVMIntegrationTestSuite) SetupSuite() {
	desc = NewServiceDesc("127.0.0.1", 19090, 10, true)

	var err error
	t.EthClient, err = web3.Dial("http://localhost:18545")
	if err != nil {
		panic(err)
	}

	t.UserWallet1, t.UserWallet2, t.ValidatorWallet1 = evmWalletSetup()
}

func (t *EVMIntegrationTestSuite) TearDownSuite() {
	desc.CloseConnection()
	t.EthClient.Close()
}

func (t *EVMIntegrationTestSuite) SetupTest() {
	t.UserWallet1.GetNonce(t.EthClient)
	t.UserWallet2.GetNonce(t.EthClient)
	t.ValidatorWallet1.GetNonce(t.EthClient)

	t.UserWallet1.CosmosWalletInfo.RefreshSequence()
	t.UserWallet2.CosmosWalletInfo.RefreshSequence()
	t.ValidatorWallet1.CosmosWalletInfo.RefreshSequence()
}

func (i *EVMIntegrationTestSuite) TeardownTest() {}

func (t *EVMIntegrationTestSuite) Test01_CheckBalance() {
	resp, err := t.EthClient.BalanceAt(context.Background(), t.UserWallet1.EthAddress, nil)

	assert.NoError(t.T(), err)

	expectedInt := new(big.Int)
	expectedInt, _ = expectedInt.SetString("99000000000000000000", 10)
	assert.GreaterOrEqual(t.T(), 1, resp.Cmp(expectedInt))

	expectedInt, _ = expectedInt.SetString("100000000000000000000", 10)
	assert.LessOrEqual(t.T(), -1, resp.Cmp(expectedInt))
}

func (t *EVMIntegrationTestSuite) Test02_DeployTokenContract() {
	// Prepare parameters
	networkId, err := t.EthClient.NetworkID(context.Background())
	assert.NoError(t.T(), err)

	ethPrivkey, _ := ethcrypto.ToECDSA(t.UserWallet1.CosmosWalletInfo.PrivKey.Bytes())
	auth, err := abibind.NewKeyedTransactorWithChainID(ethPrivkey, networkId)
	assert.NoError(t.T(), err)

	auth.GasLimit = uint64(1300000)
	auth.GasPrice, _ = new(big.Int).SetString(xplaGasPrice, 10)

	strbin, err := os.ReadFile(filepath.Join(".", "misc", "token.sol.bin"))
	assert.NoError(t.T(), err)

	binbyte, _ := hex.DecodeString(string(strbin))

	parsedAbi, err := TokenInterfaceMetaData.GetAbi()
	assert.NoError(t.T(), err)
	assert.NotNil(t.T(), parsedAbi)

	// Actual deploy
	address, tx, _, err := abibind.DeployContract(auth, *parsedAbi, binbyte, t.EthClient, "Example Token", "XPLAERC")
	assert.NoError(t.T(), err)
	fmt.Println("Tx hash: ", tx.Hash().String())

	time.Sleep(time.Second*blocktime + 1)

	fmt.Println("Token address: ", address.String())
	t.TokenAddress = address
}

func (t *EVMIntegrationTestSuite) Test03_ExecuteTokenContractAndQueryOnEvmJsonRpc() {
	// Prepare parameters
	networkId, err := t.EthClient.NetworkID(context.Background())
	assert.NoError(t.T(), err)

	store, err := NewTokenInterface(t.TokenAddress, t.EthClient)
	assert.NoError(t.T(), err)

	// 10^18
	multiplier, _ := new(big.Int).SetString("1000000000000000000", 10)
	amt := new(big.Int).Mul(big.NewInt(10), multiplier)

	ethPrivkey, _ := ethcrypto.ToECDSA(t.UserWallet1.CosmosWalletInfo.PrivKey.Bytes())
	auth, err := abibind.NewKeyedTransactorWithChainID(ethPrivkey, networkId)
	assert.NoError(t.T(), err)

	auth.GasLimit = uint64(300000)
	auth.GasPrice, _ = new(big.Int).SetString(xplaGasPrice, 10)

	// try to transfer
	tx, err := store.Transfer(auth, t.UserWallet2.EthAddress, amt)
	assert.NoError(t.T(), err)
	fmt.Println("Sent as ", tx.Hash().String())

	time.Sleep(time.Second*blocktime + 1)

	// query & assert
	callOpt := &abibind.CallOpts{}
	resp, err := store.BalanceOf(callOpt, t.UserWallet2.EthAddress)
	assert.NoError(t.T(), err)

	assert.Equal(t.T(), amt, resp)
}

// Wrote and tried to test triggering EVM by MsgEthereumTx
// But there is a collision between tx msg caching <> ethermint antehandler
// MsgEthereumTx.From kept left <> ethermint antehandler checks and passes only MsgEthereumTx.From is empty
// It resolves from ethermint v0.20
// Before that, EVM can only be triggered by 8545

// func (t *EVMIntegrationTestSuite) Test04_ExecuteTokenContractAndQueryOnCosmos() {
// 	store, err := NewTokenInterface(t.TokenAddress, t.EthClient)
// 	assert.NoError(t.T(), err)

// 	networkId, err := t.EthClient.NetworkID(context.Background())
// 	assert.NoError(t.T(), err)

// 	multiplier, _ := new(big.Int).SetString("1000000000000000000", 10)
// 	amt := new(big.Int).Mul(big.NewInt(10), multiplier)

// 	ethPrivkey, _ := ethcrypto.ToECDSA(t.UserWallet1.CosmosWalletInfo.PrivKey.Bytes())
// 	auth, err := abibind.NewKeyedTransactorWithChainID(ethPrivkey, networkId)
// 	assert.NoError(t.T(), err)

// 	auth.NoSend = true
// 	auth.GasLimit = uint64(xplaGeneralGasLimit)
// 	auth.GasPrice, _ = new(big.Int).SetString(xplaGasPrice, 10)

// 	unsentTx, err := store.Transfer(auth, t.UserWallet2.EthAddress, amt)
// 	assert.NoError(t.T(), err)

// 	msg := &evmtypes.MsgEthereumTx{}
// 	err = msg.FromEthereumTx(unsentTx)
// 	assert.NoError(t.T(), err)

// 	feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
// 	fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

// 	txHash, err := t.UserWallet1.CosmosWalletInfo.SendTx(ChainID, msg, fee, xplaGeneralGasLimit, true)
// 	assert.NoError(t.T(), err)

// 	err = txCheck(txHash)
// 	assert.NoError(t.T(), err)

// 	// check
// 	callOpt := &abibind.CallOpts{}
// 	resp, err := store.BalanceOf(callOpt, t.UserWallet2.EthAddress)
// 	assert.NoError(t.T(), err)

// 	fmt.Println(resp.String())

// 	assert.Equal(t.T(), new(big.Int).Add(amt, amt), resp)
// }
