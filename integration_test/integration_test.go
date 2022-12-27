package integrationtest

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	wasmtype "github.com/CosmWasm/wasmd/x/wasm/types"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	banktype "github.com/cosmos/cosmos-sdk/x/bank/types"
	govtype "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtype "github.com/cosmos/cosmos-sdk/x/staking/types"

	// evmtypes "github.com/evmos/ethermint/x/evm/types"

	abibind "github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	web3 "github.com/ethereum/go-ethereum/ethclient"
	xatptype "github.com/xpladev/xpla/x/xatp/types"

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

	FactoryAddress string
	TokenAddress   string
	PairAddress    string

	ProposalId uint64

	UserWallet1      *WalletInfo
	UserWallet2      *WalletInfo
	ValidatorWallet1 *WalletInfo
	ValidatorWallet2 *WalletInfo
	ValidatorWallet3 *WalletInfo
	ValidatorWallet4 *WalletInfo
}

func (i *WASMIntegrationTestSuite) SetupSuite() {
	desc = NewServiceDesc("127.0.0.1", 9090, 10, true)

	i.UserWallet1, i.UserWallet2, i.ValidatorWallet1, i.ValidatorWallet2, i.ValidatorWallet3, i.ValidatorWallet4 = walletSetup()
}

func (i *WASMIntegrationTestSuite) SetupTest() {
	i.UserWallet1, i.UserWallet2, i.ValidatorWallet1, i.ValidatorWallet2, i.ValidatorWallet3, i.ValidatorWallet4 = walletSetup()

	i.UserWallet1.RefreshSequence()
	i.UserWallet2.RefreshSequence()
	i.ValidatorWallet1.RefreshSequence()

	fmt.Println("======== Starting", i.T().Name(), "... ========")
}

func (i *WASMIntegrationTestSuite) TearDownTest() {
	fmt.Println("======== Finished", i.T().Name(), "... ========")
}

func (u *WASMIntegrationTestSuite) TearDownSuite() {
	desc.CloseConnection()
}

// Test strategy
// 1. Simple delegation
// 2. Contract upload
// 3. Contract initiate
// 4. Contract execute

func (t *WASMIntegrationTestSuite) Test01_SimpleDelegation() {
	fmt.Println("Preparing delegation message")

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

	if assert.Equal(t.T(), expected, delegatedList) {
		fmt.Println("Delegation confirmed")
	}
}

func (t *WASMIntegrationTestSuite) Test02_StoreCode() {
	fmt.Println("Preparing code storing message")

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

	queryClient := wasmtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryCodeMsg := &wasmtype.QueryCodeRequest{
		CodeId: 1,
	}

	resp, err := queryClient.Code(context.Background(), queryCodeMsg)

	if assert.NoError(t.T(), err) && assert.NotNil(t.T(), resp) {
		fmt.Println("Code storage confirmed:", resp.CodeID)
	}
}

func (t *WASMIntegrationTestSuite) Test03_InstantiateContract() {
	fmt.Println("Preparing code instantiating message")

	initMsg := []byte(fmt.Sprintf(`
		{
			"name": "testtoken",
			"symbol": "TKN",
			"decimals": 6,
			"initial_balances": [
				{
					"address": "%s",
					"amount": "1000000000"
				},
				{
					"address": "%s",
					"amount": "1000000000"
				}
			]
		}
	`, t.UserWallet2.StringAddress, t.ValidatorWallet1.StringAddress))

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

ATTR:
	for _, val := range resp.TxResponse.Events {
		for _, attr := range val.Attributes {
			if string(attr.Key) == "_contract_address" {
				t.TokenAddress = string(attr.Value)
				break ATTR
			}
		}
	}

	fmt.Println("Token is deployed as", t.TokenAddress)

	fmt.Println("Checking the token balance")

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
	if assert.NoError(t.T(), err) && assert.Equal(t.T(), "1000000000", amtResp.Balance) {
		fmt.Println("Token balance confirmed")
	}
}

func (t *WASMIntegrationTestSuite) Test04_ContractExecution() {
	fmt.Println("Preparing contract execution message")

	transferMsg := []byte(fmt.Sprintf(`
		{
			"transfer": {
				"recipient": "%s",
				"amount": "500000000"
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

	fmt.Println("Checking the token balance")

	// Balance check after transfer
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

	if assert.NoError(t.T(), err) && assert.Equal(t.T(), "500000000", amtResp.Balance) {
		fmt.Println("Token balance confirmed")
	}
}

func (t *WASMIntegrationTestSuite) Test05_EnableSwap() {
	// WASM code store
	{
		for _, filename := range []string{
			"pair.wasm",    // Code: 2
			"factory.wasm", // Code: 3
		} {
			fmt.Println("Preparing code storing message:", filename)

			contractBytes, err := os.ReadFile(filepath.Join(".", "misc", filename))
			if err != nil {
				panic(err)
			}

			storeMsg := &wasmtype.MsgStoreCode{
				Sender:       t.UserWallet1.StringAddress,
				WASMByteCode: contractBytes,
			}

			feeAmt := sdktypes.NewDec(xplaPairInstantiateGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))

			fee := sdktypes.Coin{
				Denom:  "axpla",
				Amount: feeAmt.Ceil().RoundInt(),
			}

			txhash, err := t.UserWallet1.SendTx(ChainID, storeMsg, fee, xplaCodeGasLimit, false)

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
	}

	// factory instantiate
	{
		fmt.Println("Preparing factory code instantiate")

		initMsg := []byte(`{
			"pair_code_id": 2,
			"token_code_id": 1
		}`)

		instantiateMsg := &wasmtype.MsgInstantiateContract{
			Sender: t.UserWallet1.StringAddress,
			Admin:  t.UserWallet1.StringAddress,
			CodeID: 3,
			Label:  "Integration test purpose",
			Msg:    initMsg,
		}

		feeAmt := sdktypes.NewDec(xplaPairInstantiateGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.UserWallet1.SendTx(ChainID, instantiateMsg, fee, xplaPairInstantiateGasLimit, false)
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

	ATTR1:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "_contract_address" {
					t.FactoryAddress = string(attr.Value)
					break ATTR1
				}
			}
		}

		fmt.Println("Factory contract is instantiated as", t.FactoryAddress)
	}

	{
		fmt.Println("Registering axpla")

		registerMsg := []byte(`
			{
				"add_native_token_decimals": {
				"denom": "axpla",
				"decimals": 18
				}
			}
		`)

		contractExecMsg := &wasmtype.MsgExecuteContract{
			Sender:   t.UserWallet1.StringAddress,
			Contract: t.FactoryAddress,
			Msg:      registerMsg,
			Funds:    sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(1))),
		}

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.UserWallet1.SendTx(ChainID, contractExecMsg, fee, xplaGeneralGasLimit, false)
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

	// create a pair from the factory
	{
		fmt.Println("Creating a pair using the factory contract")

		createPairMsg := []byte(fmt.Sprintf(`
			{
				"create_pair": {
					"asset_infos": [
						{
							"token": {
								"contract_addr": "%s"
							}
						},
						{
							"native_token": {
								"denom": "axpla"
							}
						}
					]
				}
			}
		`, t.TokenAddress))

		contractExecMsg := &wasmtype.MsgExecuteContract{
			Sender:   t.UserWallet1.StringAddress,
			Contract: t.FactoryAddress,
			Msg:      createPairMsg,
		}

		feeAmt := sdktypes.NewDec(xplaCreatePairGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.UserWallet1.SendTx(ChainID, contractExecMsg, fee, xplaCreatePairGasLimit, false)
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

	ATTR2:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "pair_contract_addr" {
					t.PairAddress = string(attr.Value)
					break ATTR2
				}
			}
		}

		fmt.Println("Pair contract is instantiated as", t.PairAddress)
	}

	// increase allowance
	// provide initial liquidity
	{
		fmt.Println("Providing the token to the pair")

		increaseAllowanceMsg := []byte(fmt.Sprintf(`
			{
				"increase_allowance": {
					"spender": "%s",
					"amount": "100000000",
					"expires": {
						"never": {}
					}
				}
			}
		`, t.PairAddress))

		increaseAllowanceExecMsg := &wasmtype.MsgExecuteContract{
			Sender:   t.UserWallet1.StringAddress,
			Contract: t.TokenAddress,
			Msg:      increaseAllowanceMsg,
		}

		feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := t.UserWallet1.SendTx(ChainID, increaseAllowanceExecMsg, fee, xplaGeneralGasLimit, false)
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

		provideMsg := []byte(fmt.Sprintf(`
			{
				"provide_liquidity": {
					"assets": [
						{
							"info" : {
								"token": {
									"contract_addr": "%s"
								}
							},
							"amount": "100000000"
						},
						{
							"info" : {
								"native_token": {
									"denom": "axpla"
								}
							},
							"amount": "10000000000000000000"
						}
					]
				}
		  	}
		`, t.TokenAddress)) // 100 TKN : 10 XPLA

		provideExecMsg := &wasmtype.MsgExecuteContract{
			Sender:   t.UserWallet1.StringAddress,
			Contract: t.PairAddress,
			Msg:      provideMsg,
			Funds:    sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.MustNewDecFromStr("10000000000000000000").RoundInt())),
		}

		feeAmt = sdktypes.NewDec(xplaPairGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee = sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err = t.UserWallet1.SendTx(ChainID, provideExecMsg, fee, xplaPairGasLimit, false)
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
}

func (t *WASMIntegrationTestSuite) Test06_EnableXATP() {
	// submit proposal
	{
		fmt.Println("Raising a proposal with depositing the token")

		proposalContent := xatptype.NewRegisterXatpProposal("xatp test", "test desc", t.TokenAddress, t.PairAddress, "TKN", 6)

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

		txhash, err := t.UserWallet1.SendTx(fee, xplaProposalGasLimit, proposalMsg)
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

	RAISE:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "proposal_id" {
					t.ProposalId, _ = strconv.ParseUint(string(attr.Value), 10, 64)
					break RAISE
				}
			}
		}

		fmt.Println("Proposal is proposed as ID", t.ProposalId)
	}

	// voting
	{
		fmt.Println("Voting to the proposal")

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

				voteMsg := govtype.NewMsgVote(addr.ByteAddress, t.ProposalId, govtype.OptionYes)
				feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
				fee := sdktypes.Coin{
					Denom:  "axpla",
					Amount: feeAmt.Ceil().RoundInt(),
				}

				txhash, err := addr.SendTx(fee, xplaGeneralGasLimit, voteMsg)
				if assert.NotEqual(t.T(), "", txhash) && assert.NoError(t.T(), err) {
					fmt.Println(addr.StringAddress, "voted to the proposal", t.ProposalId, "as tx", txhash, "err:", err)
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

	fmt.Println("Waiting for 20 sec for passing the proposal...")
	time.Sleep(time.Second * 20)

	// check
	{
		fmt.Println("XATP status check")

		client := xatptype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

		resp, err := client.Xatp(context.Background(), &xatptype.QueryXatpRequest{Denom: "TKN"})

		assert.NoError(t.T(), err)
		assert.NotNil(t.T(), resp)

		expectedXATP := xatptype.XATP{
			Denom:    "TKN",
			Token:    t.TokenAddress,
			Pair:     t.PairAddress,
			Decimals: 6,
		}

		if assert.Equal(t.T(), expectedXATP, resp.GetXatp()) {
			fmt.Println("XATP status check done")
		}
	}
}

func (t *WASMIntegrationTestSuite) Test07_NativeTransfer_PayByXPLA_LackOfFee_SimulationShouldFail() {
	fmt.Println("Preparing transfer message")

	trasnferMsg := banktype.NewMsgSend(
		t.ValidatorWallet1.ByteAddress, t.ValidatorWallet2.ByteAddress,
		sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(1_000000_000000_000000))),
	)

	fmt.Println("Pay very low fee. Should fail")
	feeAmt := sdktypes.NewDec(xplaLowGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
	fee := sdktypes.Coin{
		Denom:  "axpla", // ordinal native fee
		Amount: feeAmt.Ceil().RoundInt(),
	}

	txhash, err := t.ValidatorWallet1.SendTx(ChainID, trasnferMsg, fee, xplaGeneralGasLimit, false)
	if assert.Error(t.T(), err, "should fail in the simulation step") && assert.Equal(t.T(), "", txhash) {
		fmt.Println("Error successfully raised", err)
	} else {
		fmt.Println("Tx hash detected:", txhash)
		fmt.Println("No error is detected on simulation phase. Test fail")
	}

	err = txCheck(txhash)
	if assert.Error(t.T(), err) {
		fmt.Println("Error confirmed after broadcasted:", err)
	}
}

func (t *WASMIntegrationTestSuite) Test08_TxFeePaidByCw20_NativeTransfer_LackOfFee_SimulationShouldFail() {
	fmt.Println("Preparing transfer message")

	trasnferMsg := banktype.NewMsgSend(
		t.ValidatorWallet1.ByteAddress, t.ValidatorWallet2.ByteAddress,
		sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(1_000000_000000_000000))),
	)

	fmt.Println("Pay very low fee. Should fail")
	feeAmt := sdktypes.NewDec(xplaLowGasLimit).Mul(sdktypes.MustNewDecFromStr(tknGasPrice))
	fee := sdktypes.Coin{
		Denom:  "TKN", // use XATP
		Amount: feeAmt.Ceil().RoundInt(),
	}

	txhash, err := t.ValidatorWallet1.SendTx(fee, xplaPairGasLimit, trasnferMsg)
	if assert.Error(t.T(), err, "should fail in the simulation step") && assert.Equal(t.T(), "", txhash) {
		fmt.Println("Error successfully raised", err)
	} else {
		fmt.Println("Tx hash detected:", txhash)
		fmt.Println("No error is detected on simulation phase. Test fail")
	}

	err = txCheck(txhash)
	if assert.Error(t.T(), err) {
		fmt.Println("Error confirmed after broadcasted:", err)
	}
}

func (t *WASMIntegrationTestSuite) Test09_TxFeePaidByCw20_NativeTransfer() {
	fmt.Println("Preparing transfer message")

	trasnferMsg := banktype.NewMsgSend(
		t.ValidatorWallet1.ByteAddress, t.ValidatorWallet2.ByteAddress,
		sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(1_000000_000000_000000))),
	)

	fmt.Println("Pay fee as 0.068 TKN")
	feeAmt := sdktypes.NewDec(xplaPairGasLimit).Mul(sdktypes.MustNewDecFromStr(tknGasPrice))
	fee := sdktypes.Coin{
		Denom:  "TKN",
		Amount: feeAmt.Ceil().RoundInt(),
	}

	txhash, err := t.ValidatorWallet1.SendTx(ChainID, trasnferMsg, fee, xplaGeneralGasLimit, false)
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

func (t *WASMIntegrationTestSuite) Test10_TxFeePaidByCw20_Cw20Transfer() {
	fmt.Println("Preparing contract execution message")

	transferMsg := []byte(fmt.Sprintf(`
		{
			"transfer": {
				"recipient": "%s",
				"amount": "100000000"
			}
		}
	`, t.UserWallet1.StringAddress))

	contractExecMsg := &wasmtype.MsgExecuteContract{
		Sender:   t.UserWallet2.StringAddress,
		Contract: t.TokenAddress,
		Msg:      transferMsg,
	}

	fmt.Println("Pay fee as 0.0765 TKN")
	feeAmt := sdktypes.NewDec(xplaXATPTransferGasLimit).Mul(sdktypes.MustNewDecFromStr(tknGasPrice))
	fee := sdktypes.Coin{
		Denom:  "TKN",
		Amount: feeAmt.Ceil().RoundInt(),
	}

	txhash, err := t.UserWallet2.SendTx(ChainID, contractExecMsg, fee, xplaXATPTransferGasLimit, false)
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

	fmt.Println("Checking the token balance")

	queryTokenTransferWithXaptClient := wasmtype.NewQueryClient(desc.GetConnectionWithContext(context.Background()))

	queryStr := []byte(fmt.Sprintf(`{
		"balance": {
			"address": "%s"
		}
	}`, t.UserWallet1.StringAddress))

	tokenResp, err := queryTokenTransferWithXaptClient.SmartContractState(context.Background(), &wasmtype.QuerySmartContractStateRequest{
		Address:   t.TokenAddress,
		QueryData: queryStr,
	})

	if !(assert.NoError(t.T(), err) && assert.NotNil(t.T(), tokenResp)) {
		fmt.Println("Err in the contract querying")
	}

	type AmtResp struct {
		Balance string `json:"balance"`
	}

	amtXATPResp := &AmtResp{}
	err = json.Unmarshal(tokenResp.Data.Bytes(), amtXATPResp)
	if assert.NoError(t.T(), err) && assert.Equal(t.T(), "500000000", amtXATPResp.Balance) {
		fmt.Println("Token balance confirmed")
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
	desc = NewServiceDesc("127.0.0.1", 9090, 10, true)

	var err error
	t.EthClient, err = web3.Dial("http://localhost:8545")
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

func walletSetup() (userWallet1, userWallet2, validatorWallet1, validatorWallet2, validatorWallet3, validatorWallet4 *WalletInfo) {
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
	var resp *txtypes.GetTxResponse

	txClient := txtypes.NewServiceClient(desc.GetConnectionWithContext(context.Background()))

	for i := 0; i < 20; i++ {
		resp, err = txClient.GetTx(context.Background(), &txtypes.GetTxRequest{Hash: txHash})

		if err == nil {
			if resp.TxResponse.Code != 0 {
				return errors.New(resp.TxResponse.RawLog)
			}

			return nil
		}

		time.Sleep(time.Second / 2)
	}

	return err
}
