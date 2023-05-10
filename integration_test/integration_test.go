package integrationtest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	wasmtype "github.com/CosmWasm/wasmd/x/wasm/types"
	tmservicetypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	ed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	govtype "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtype "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"

	zrValidatorType "github.com/xpladev/xpla/x/zeroreward/types"

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

	UserWallet1                *WalletInfo
	UserWallet2                *WalletInfo
	ValidatorWallet1           *WalletInfo
	ValidatorWallet2           *WalletInfo
	ValidatorWallet3           *WalletInfo
	ValidatorWallet4           *WalletInfo
	ValidatorWallet5           *WalletInfo
	ZeroRewardValidatorWallet1 *WalletInfo
	ZeroRewardValidatorWallet2 *WalletInfo
	ZeroRewardValidatorWallet3 *WalletInfo

	ZeroRewardValidatorPVKey1 *PVKey
	ZeroRewardValidatorPVKey2 *PVKey
	ZeroRewardValidatorPVKey3 *PVKey
	Validator5PVKey           *PVKey

	GeneralZeroRewardValidatorRegistrationProposalID uint64
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
		i.ZeroRewardValidatorWallet1,
		i.ZeroRewardValidatorWallet2,
		i.ZeroRewardValidatorWallet3 = walletSetup()
}

func (i *WASMIntegrationTestSuite) SetupTest() {
	i.UserWallet1,
		i.UserWallet2,
		i.ValidatorWallet1,
		i.ValidatorWallet2,
		i.ValidatorWallet3,
		i.ValidatorWallet4,
		i.ValidatorWallet5,
		i.ZeroRewardValidatorWallet1,
		i.ZeroRewardValidatorWallet2,
		i.ZeroRewardValidatorWallet3 = walletSetup()

	i.UserWallet1.RefreshSequence()
	i.UserWallet2.RefreshSequence()
	i.ValidatorWallet1.RefreshSequence()
	i.ValidatorWallet2.RefreshSequence()
	i.ValidatorWallet3.RefreshSequence()
	i.ValidatorWallet4.RefreshSequence()
	i.ValidatorWallet5.RefreshSequence()
	i.ZeroRewardValidatorWallet1.RefreshSequence()

	var err error
	i.ZeroRewardValidatorPVKey1, err = loadPrivValidator("zeroreward_validator1")
	if err != nil {
		i.Fail("PVKey load fail - 1")
	}

	i.ZeroRewardValidatorPVKey2, err = loadPrivValidator("zeroreward_validator2")
	if err != nil {
		i.Fail("PVKey load fail - 2")
	}

	i.ZeroRewardValidatorPVKey3, err = loadPrivValidator("zeroreward_validator3")
	if err != nil {
		i.Fail("PVKey load fail - 3")
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
	coin := &sdktypes.Coin{
		Denom:  "axpla",
		Amount: amt,
	}

	delegationMsg := stakingtype.NewMsgDelegate(
		t.UserWallet1.ByteAddress,
		t.ValidatorWallet1.ByteAddress.Bytes(),
		*coin,
	)

	feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.Coin{
		Denom:  "axpla",
		Amount: feeAmt.Ceil().RoundInt(),
	}

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
			sdktypes.Coin{
				Denom:  "axpla",
				Amount: amt,
			},
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

	fee := sdktypes.Coin{
		Denom:  "axpla",
		Amount: feeAmt.Ceil().RoundInt(),
	}

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
	fee := sdktypes.Coin{
		Denom:  "axpla",
		Amount: feeAmt.Ceil().RoundInt(),
	}

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
	fee := sdktypes.Coin{
		Denom:  "axpla",
		Amount: feeAmt.Ceil().RoundInt(),
	}

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

func (t *WASMIntegrationTestSuite) Test12_GeneralZeroRewardValidatorRegistryUnregistryDelegation() {
	amt := sdktypes.NewInt(1000000000000000000)

	{
		fmt.Println("Preparing proposal to add a zero reward validator")

		proposalContent, err := zrValidatorType.NewRegisterZeroRewardValidatorProposal(
			"register_zeroreward_validator",
			"Test zero reward validator registary",
			sdktypes.AccAddress(t.ZeroRewardValidatorWallet1.ByteAddress.Bytes()),
			sdktypes.ValAddress(t.ZeroRewardValidatorWallet1.ByteAddress.Bytes()),
			&ed25519.PubKey{Key: t.ZeroRewardValidatorPVKey1.PubKey.Bytes()},
			sdktypes.NewCoin("axpla", amt), // smaller amount than other basic validators
			stakingtype.NewDescription("zeroreward_validator", "", "", "", ""),
		)

		assert.NoError(t.T(), err)

		proposalMsg, err := govtype.NewMsgSubmitProposal(
			proposalContent,
			sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
			t.ZeroRewardValidatorWallet1.ByteAddress,
		)

		assert.NoError(t.T(), err)

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.ZeroRewardValidatorWallet1.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
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

		queryClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
		resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
			Hash: txhash,
		})

		assert.NoError(t.T(), err)

	GENERAL_ZEROREWARD_VALIDATOR_REGISTERED:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "proposal_id" {
					t.GeneralZeroRewardValidatorRegistrationProposalID, _ = strconv.ParseUint(string(attr.Value), 10, 64)
					break GENERAL_ZEROREWARD_VALIDATOR_REGISTERED
				}
			}
		}

		fmt.Println("Proposal is applied as ID", t.GeneralZeroRewardValidatorRegistrationProposalID)
	}

	{
		fmt.Println("Vote for the zero reward validator registration")

		wg := sync.WaitGroup{}

		errChan := make(chan error)
		successChan := make(chan bool)

		for _, addr := range []*WalletInfo{
			t.ValidatorWallet1,
			t.ValidatorWallet2,
			t.ValidatorWallet3,
			t.ValidatorWallet4,
		} {
			wg.Add(1)

			go func(addr *WalletInfo) {
				defer wg.Done()

				voteMsg := govtype.NewMsgVote(addr.ByteAddress, t.GeneralZeroRewardValidatorRegistrationProposalID, govtype.OptionYes)
				feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
				fee := sdktypes.Coin{
					Denom:  "axpla",
					Amount: feeAmt.Ceil().RoundInt(),
				}

				txhash, err := addr.SendTx(ChainID, voteMsg, fee, xplaGeneralGasLimit, false)
				if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
					fmt.Println(addr.StringAddress, "voted to the proposal", t.GeneralZeroRewardValidatorRegistrationProposalID, "as tx", txhash, "err:", err)
				} else {
					fmt.Println(err)
				}

				err = txCheck(txhash)
				if assert.NoError(t.T(), err) {
					fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
				} else {
					fmt.Println(err)
				}
			}(addr)
		}

		go func() {
			wg.Wait()
			successChan <- true
		}()

	VOTE:
		for {
			select {
			case chanErr := <-errChan:
				fmt.Print(chanErr.Error())
				t.T().Fail()

				return
			case <-successChan:
				break VOTE
			}
		}
	}

	fmt.Println("Waiting 25sec for the proposal passing...")
	time.Sleep(time.Second * 25)

	{
		fmt.Println("Validator status check")

		client := tmservicetypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.GetLatestValidatorSet(context.Background(), &tmservicetypes.GetLatestValidatorSetRequest{})
		assert.NoError(t.T(), err)

		isFound := false
		conAddress := sdktypes.ConsAddress(t.ZeroRewardValidatorPVKey1.Address).String()
		for _, unitVal := range validatorStatus.Validators {
			if conAddress == unitVal.GetAddress() {
				fmt.Println("power:", unitVal.VotingPower)
				isFound = true
				break
			}
		}

		if assert.True(t.T(), isFound) {
			fmt.Println("Consensus address", conAddress, "found in validator set!")
		} else {
			fmt.Println(conAddress, "not found")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Delegator status check")

		queryClient := stakingtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

		queryDelegatorMsg := &stakingtype.QueryDelegatorDelegationsRequest{
			DelegatorAddr: t.ZeroRewardValidatorWallet1.StringAddress,
		}

		delegationResp, err := queryClient.DelegatorDelegations(context.Background(), queryDelegatorMsg)
		assert.NoError(t.T(), err)

		delegatedList := delegationResp.GetDelegationResponses()

		expected := []stakingtype.DelegationResponse{
			stakingtype.NewDelegationResp(
				t.ZeroRewardValidatorWallet1.ByteAddress,
				t.ZeroRewardValidatorWallet1.ByteAddress.Bytes(),
				sdktypes.NewDecFromInt(amt),
				sdktypes.Coin{
					Denom:  "axpla",
					Amount: amt,
				},
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

		coin := &sdktypes.Coin{
			Denom:  "axpla",
			Amount: delegationAmt,
		}

		delegationMsg := stakingtype.NewMsgDelegate(
			t.UserWallet1.ByteAddress,
			t.ZeroRewardValidatorWallet1.ByteAddress.Bytes(),
			*coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.UserWallet1.SendTx(ChainID, delegationMsg, fee, xplaGeneralGasLimit, false)
		if assert.Error(t.T(), err) && assert.Equal(t.T(), txhash, "") {
			fmt.Println("Expected failure is occurred.")
		} else {
			fmt.Println("Tx sent. Test fail")
			t.T().Fail()
		}

		// err = txCheck(txhash)
		// assert.NoError(t.T(), err)
	}

	{
		fmt.Println("Try redelegation and should fail...")

		coin := &sdktypes.Coin{
			Denom:  "axpla",
			Amount: delegationAmt,
		}

		redelegationMsg := stakingtype.NewMsgBeginRedelegate(
			t.UserWallet1.ByteAddress,
			t.ValidatorWallet1.ByteAddress.Bytes(),
			t.ZeroRewardValidatorWallet1.ByteAddress.Bytes(),
			*coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

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

		coin := &sdktypes.Coin{
			Denom:  "axpla",
			Amount: delegationAmt,
		}

		redelegationMsg := stakingtype.NewMsgUndelegate(
			t.ZeroRewardValidatorWallet1.ByteAddress.Bytes(),
			t.ZeroRewardValidatorWallet1.ByteAddress.Bytes(),
			*coin,
		)

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.ZeroRewardValidatorWallet1.SendTx(ChainID, redelegationMsg, fee, xplaGeneralGasLimit, false)
		if assert.Error(t.T(), err) && assert.Equal(t.T(), txhash, "") {
			fmt.Println("Expected failure is occurred.")
		} else {
			fmt.Println("Tx sent. Test fail")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Preparing proposal to remove a zero reward validator")

		proposalContent := zrValidatorType.NewUnregisterZeroRewardValidatorProposal(
			"unregister_zero_reward_validator",
			"Test zero reward validator unregistration",
			sdktypes.ValAddress(t.ZeroRewardValidatorWallet1.ByteAddress.Bytes()),
		)

		proposalMsg, err := govtype.NewMsgSubmitProposal(
			proposalContent,
			sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
			t.ZeroRewardValidatorWallet1.ByteAddress,
		)

		assert.NoError(t.T(), err)

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.ZeroRewardValidatorWallet1.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
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

		queryClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
		resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
			Hash: txhash,
		})

		assert.NoError(t.T(), err)

	GENERAL_ZEROREWARD_VALIDATOR_UNREGISTERED:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "proposal_id" {
					t.GeneralZeroRewardValidatorRegistrationProposalID, _ = strconv.ParseUint(string(attr.Value), 10, 64)
					break GENERAL_ZEROREWARD_VALIDATOR_UNREGISTERED
				}
			}
		}

		fmt.Println("Proposal is applied as ID", t.GeneralZeroRewardValidatorRegistrationProposalID)
	}

	{
		fmt.Println("Vote for the zero reward validator unregistration")

		wg := sync.WaitGroup{}

		errChan := make(chan error)
		successChan := make(chan bool)

		for _, addr := range []*WalletInfo{
			t.ValidatorWallet1,
			t.ValidatorWallet2,
			t.ValidatorWallet3,
			t.ValidatorWallet4,
		} {
			wg.Add(1)

			go func(addr *WalletInfo) {
				defer wg.Done()

				voteMsg := govtype.NewMsgVote(addr.ByteAddress, t.GeneralZeroRewardValidatorRegistrationProposalID, govtype.OptionYes)
				feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
				fee := sdktypes.Coin{
					Denom:  "axpla",
					Amount: feeAmt.Ceil().RoundInt(),
				}

				txhash, err := addr.SendTx(ChainID, voteMsg, fee, xplaGeneralGasLimit, false)
				if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
					fmt.Println(addr.StringAddress, "voted to the proposal", t.GeneralZeroRewardValidatorRegistrationProposalID, "as tx", txhash, "err:", err)
				} else {
					fmt.Println(err)
				}

				err = txCheck(txhash)
				if assert.NoError(t.T(), err) {
					fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
				} else {
					fmt.Println(err)
				}
			}(addr)
		}

		go func() {
			wg.Wait()
			successChan <- true
		}()

	VOTE_UNREGISTER:
		for {
			select {
			case chanErr := <-errChan:
				fmt.Print(chanErr.Error())
				t.T().Fail()

				return
			case <-successChan:
				break VOTE_UNREGISTER
			}
		}
	}

	fmt.Println("Waiting 25sec for the proposal passing...")
	time.Sleep(time.Second * 25)

	{
		fmt.Println("Check existence of the zero reward validator")

		client := zrValidatorType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.ZeroRewardValidators(context.Background(), &zrValidatorType.QueryZeroRewardValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVoluntaryValAddress := sdktypes.ValAddress(t.ZeroRewardValidatorWallet1.ByteAddress).String()

		if assert.NotContains(t.T(), validatorStatus.GetZeroRewardValidators(), thisVoluntaryValAddress) {
			fmt.Println(thisVoluntaryValAddress, "is successfully removed from validator set!")
		} else {
			fmt.Println(thisVoluntaryValAddress, "still found")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Try deregister a validator but it is not registered...")

		proposalContent := zrValidatorType.NewUnregisterZeroRewardValidatorProposal(
			"false_unregister_zero_reward_validator",
			"False zero reward validator unregistration",
			sdktypes.ValAddress(t.UserWallet1.ByteAddress.Bytes()),
		)

		proposalMsg, err := govtype.NewMsgSubmitProposal(
			proposalContent,
			sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
			t.UserWallet1.ByteAddress,
		)

		assert.NoError(t.T(), err)

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.ZeroRewardValidatorWallet1.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
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
	proposalIds := []uint64{}

	{
		fmt.Println("Preparing multiple proposals to add a zero reward validator")

		var eg errgroup.Group

		for i := 0; i < 2; i++ {
			var proposalId uint64 = 0

			eg.Go(func() error {
				{
					// apply proposal
					proposalContent, err := zrValidatorType.NewRegisterZeroRewardValidatorProposal(
						fmt.Sprintf("register_multiple_zeroreward_validator_%d", i),
						fmt.Sprintf("Test zero reward validator registary_%d", i),
						sdktypes.AccAddress(t.ZeroRewardValidatorWallet2.ByteAddress.Bytes()),
						sdktypes.ValAddress(t.ZeroRewardValidatorWallet2.ByteAddress.Bytes()),
						&ed25519.PubKey{Key: t.ZeroRewardValidatorPVKey2.PubKey.Bytes()},
						sdktypes.NewCoin("axpla", amt), // smaller amount than other basic validators
						stakingtype.NewDescription("zeroreward_validator", "", "", "", ""),
					)

					assert.NoError(t.T(), err)

					proposalMsg, err := govtype.NewMsgSubmitProposal(
						proposalContent,
						sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
						t.ZeroRewardValidatorWallet2.ByteAddress,
					)

					assert.NoError(t.T(), err)

					feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
					fee := sdktypes.Coin{
						Denom:  "axpla",
						Amount: feeAmt.Ceil().RoundInt(),
					}

					txhash, err := t.ZeroRewardValidatorWallet2.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
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

					queryClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
					resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
						Hash: txhash,
					})

					assert.NoError(t.T(), err)

				GENERAL_ZEROREWARD_VALIDATOR_REGISTERED:
					for _, val := range resp.TxResponse.Events {
						for _, attr := range val.Attributes {
							if string(attr.Key) == "proposal_id" {
								proposalId, _ = strconv.ParseUint(string(attr.Value), 10, 64)
								break GENERAL_ZEROREWARD_VALIDATOR_REGISTERED
							}
						}
					}

					proposalIds = append(proposalIds, proposalId)

					fmt.Println("Proposal is applied as ID", proposalId)
				}

				{
					// vote to the proposal

					wg := sync.WaitGroup{}

					errChan := make(chan error)
					successChan := make(chan bool)

					for _, addr := range []*WalletInfo{
						t.ValidatorWallet1,
						t.ValidatorWallet2,
						t.ValidatorWallet3,
						t.ValidatorWallet4,
					} {
						wg.Add(1)

						go func(addr *WalletInfo) {
							defer wg.Done()

							voteMsg := govtype.NewMsgVote(addr.ByteAddress, proposalId, govtype.OptionYes)
							feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
							fee := sdktypes.Coin{
								Denom:  "axpla",
								Amount: feeAmt.Ceil().RoundInt(),
							}

							txhash, err := addr.SendTx(ChainID, voteMsg, fee, xplaGeneralGasLimit, false)
							if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
								fmt.Println(addr.StringAddress, "voted to the proposal", proposalId, "as tx", txhash, "err:", err)
							} else {
								fmt.Println(err)
							}

							err = txCheck(txhash)
							if assert.NoError(t.T(), err) {
								fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
							} else {
								fmt.Println(err)
							}
						}(addr)
					}

					go func() {
						wg.Wait()
						successChan <- true
					}()

				VOTE:
					for {
						select {
						case chanErr := <-errChan:
							fmt.Print(chanErr.Error())
							t.T().Fail()

							return chanErr
						case <-successChan:
							break VOTE
						}
					}
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

	fmt.Println("Waiting 30sec for the proposal passing...")
	time.Sleep(time.Second * 30)

	{
		fmt.Println("Check existence of the voluntary validator")

		client := zrValidatorType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.ZeroRewardValidators(context.Background(), &zrValidatorType.QueryZeroRewardValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVoluntaryValAddress := sdktypes.ValAddress(t.ZeroRewardValidatorWallet2.ByteAddress).String()

		if len(validatorStatus.GetZeroRewardValidators()) == 1 &&
			assert.Contains(t.T(), validatorStatus.GetZeroRewardValidators(), thisVoluntaryValAddress) {
			fmt.Println(thisVoluntaryValAddress, "successfully get in the validator set!")
		} else {
			fmt.Println(thisVoluntaryValAddress, "does not exist")
			t.T().Fail()
		}
	}

	{
		fmt.Println("Preparing multiple proposals to remove a zero reward validator")

		var eg errgroup.Group

		for i := 0; i < 2; i++ {
			var proposalId uint64 = 0

			eg.Go(func() error {
				{
					// apply proposal
					proposalContent := zrValidatorType.NewUnregisterZeroRewardValidatorProposal(
						fmt.Sprintf("unregister_multiple_zeroreward_validator_%d", i),
						fmt.Sprintf("Test zero reward validator unregistary_%d", i),
						sdktypes.ValAddress(t.ZeroRewardValidatorWallet2.ByteAddress.Bytes()),
					)

					proposalMsg, err := govtype.NewMsgSubmitProposal(
						proposalContent,
						sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
						t.ZeroRewardValidatorWallet2.ByteAddress,
					)

					assert.NoError(t.T(), err)

					feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
					fee := sdktypes.Coin{
						Denom:  "axpla",
						Amount: feeAmt.Ceil().RoundInt(),
					}

					txhash, err := t.ZeroRewardValidatorWallet2.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
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

					queryClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
					resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
						Hash: txhash,
					})

					assert.NoError(t.T(), err)

				GENERAL_ZEROREWARD_VALIDATOR_UNREGISTERED:
					for _, val := range resp.TxResponse.Events {
						for _, attr := range val.Attributes {
							if string(attr.Key) == "proposal_id" {
								proposalId, _ = strconv.ParseUint(string(attr.Value), 10, 64)
								break GENERAL_ZEROREWARD_VALIDATOR_UNREGISTERED
							}
						}
					}

					proposalIds = append(proposalIds, proposalId)

					fmt.Println("Proposal is applied as ID", proposalId)
				}

				{
					// vote to the proposal

					wg := sync.WaitGroup{}

					errChan := make(chan error)
					successChan := make(chan bool)

					for _, addr := range []*WalletInfo{
						t.ValidatorWallet1,
						t.ValidatorWallet2,
						t.ValidatorWallet3,
						t.ValidatorWallet4,
					} {
						wg.Add(1)

						go func(addr *WalletInfo) {
							defer wg.Done()

							voteMsg := govtype.NewMsgVote(addr.ByteAddress, proposalId, govtype.OptionYes)
							feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
							fee := sdktypes.Coin{
								Denom:  "axpla",
								Amount: feeAmt.Ceil().RoundInt(),
							}

							txhash, err := addr.SendTx(ChainID, voteMsg, fee, xplaGeneralGasLimit, false)
							if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
								fmt.Println(addr.StringAddress, "voted to the proposal", proposalId, "as tx", txhash, "err:", err)
							} else {
								fmt.Println(err)
							}

							err = txCheck(txhash)
							if assert.NoError(t.T(), err) {
								fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
							} else {
								fmt.Println(err)
							}
						}(addr)
					}

					go func() {
						wg.Wait()
						successChan <- true
					}()

				VOTE:
					for {
						select {
						case chanErr := <-errChan:
							fmt.Print(chanErr.Error())
							t.T().Fail()

							return chanErr
						case <-successChan:
							break VOTE
						}
					}
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

	fmt.Println("Waiting 30sec for the proposal passing...")
	time.Sleep(time.Second * 30)

	{
		fmt.Println("Check existence of the voluntary validator")

		client := zrValidatorType.NewQueryClient(desc.GetConnectionWithContext(context.Background()))
		validatorStatus, err := client.ZeroRewardValidators(context.Background(), &zrValidatorType.QueryZeroRewardValidatorsRequest{})
		assert.NoError(t.T(), err)

		thisVoluntaryValAddress := sdktypes.ValAddress(t.ZeroRewardValidatorWallet2.ByteAddress).String()

		if assert.NotContains(t.T(), validatorStatus.GetZeroRewardValidators(), thisVoluntaryValAddress) {
			fmt.Println(thisVoluntaryValAddress, "successfully removed from the validator set!")
		} else {
			fmt.Println(thisVoluntaryValAddress, "still exist!")
			t.T().Fail()
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

	time.Sleep(time.Second * 7)

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

	time.Sleep(time.Second * 7)

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
// 	fee := sdktypes.Coin{
// 		Denom:  "axpla",
// 		Amount: feeAmt.Ceil().RoundInt(),
// 	}

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

func walletSetup() (
	userWallet1, userWallet2,
	validatorWallet1, validatorWallet2, validatorWallet3, validatorWallet4, validatorWallet5,
	zeroRewardValidatorWallet1, zeroRewardValidatorWallet2, zeroRewardValidatorWallet3 *WalletInfo,
) {
	var err error

	user1Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "user1.mnemonics"))
	if err != nil {
		panic(err)
	}

	userWallet1, err = NewWalletInfo(string(user1Mnemonics))
	if err != nil {
		panic(err)
	}

	user2Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "user2.mnemonics"))
	if err != nil {
		panic(err)
	}

	userWallet2, err = NewWalletInfo(string(user2Mnemonics))
	if err != nil {
		panic(err)
	}

	validator1Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator1.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet1, err = NewWalletInfo(string(validator1Mnemonics))
	if err != nil {
		panic(err)
	}

	validator2Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator2.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet2, err = NewWalletInfo(string(validator2Mnemonics))
	if err != nil {
		panic(err)
	}

	validator3Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator3.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet3, err = NewWalletInfo(string(validator3Mnemonics))
	if err != nil {
		panic(err)
	}

	validator4Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator4.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet4, err = NewWalletInfo(string(validator4Mnemonics))
	if err != nil {
		panic(err)
	}

	validator5Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator5_experimental.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet5, err = NewWalletInfo(string(validator5Mnemonics))
	if err != nil {
		panic(err)
	}

	zerorewardValidator1, err := os.ReadFile(filepath.Join(".", "test_keys", "zeroreward_validator1.mnemonics"))
	if err != nil {
		panic(err)
	}

	zeroRewardValidatorWallet1, err = NewWalletInfo(string(zerorewardValidator1))
	if err != nil {
		panic(err)
	}

	zerorewardValidator2, err := os.ReadFile(filepath.Join(".", "test_keys", "zeroreward_validator2.mnemonics"))
	if err != nil {
		panic(err)
	}

	zeroRewardValidatorWallet2, err = NewWalletInfo(string(zerorewardValidator2))
	if err != nil {
		panic(err)
	}

	zerorewardValidator3, err := os.ReadFile(filepath.Join(".", "test_keys", "zeroreward_validator3.mnemonics"))
	if err != nil {
		panic(err)
	}

	zeroRewardValidatorWallet3, err = NewWalletInfo(string(zerorewardValidator3))
	if err != nil {
		panic(err)
	}

	return
}

func evmWalletSetup() (userWallet1, userWallet2, validatorWallet1 *EVMWalletInfo) {
	var err error

	user1Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "user1.mnemonics"))
	if err != nil {
		panic(err)
	}

	userWallet1, err = NewEVMWalletInfo(string(user1Mnemonics))
	if err != nil {
		panic(err)
	}

	user2Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "user2.mnemonics"))
	if err != nil {
		panic(err)
	}

	userWallet2, err = NewEVMWalletInfo(string(user2Mnemonics))
	if err != nil {
		panic(err)
	}

	validator1Mnemonics, err := os.ReadFile(filepath.Join(".", "test_keys", "validator1.mnemonics"))
	if err != nil {
		panic(err)
	}

	validatorWallet1, err = NewEVMWalletInfo(string(validator1Mnemonics))
	if err != nil {
		panic(err)
	}

	return
}

func txCheck(txHash string) error {
	var err error

	for i := 0; i < 20; i++ {
		txClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))
		_, err = txClient.GetTx(context.Background(), &txtypes.GetTxRequest{Hash: txHash})

		if err == nil {
			return nil
		}

		time.Sleep(time.Second / 2)
	}

	return err
}

// copied from Tendermint type
type PVKey struct {
	Address tmtypes.Address  `json:"address"`
	PubKey  tmcrypto.PubKey  `json:"pub_key"`
	PrivKey tmcrypto.PrivKey `json:"priv_key"`
}

func loadPrivValidator(validatorName string) (*PVKey, error) {
	valKeyBytes, err := os.ReadFile(filepath.Join(".", validatorName, "priv_validator_key.json"))
	if err != nil {
		return nil, err
	}

	pvKey := PVKey{}
	err = tmjson.Unmarshal(valKeyBytes, &pvKey)
	if err != nil {
		return nil, err
	}

	return &pvKey, nil
}
