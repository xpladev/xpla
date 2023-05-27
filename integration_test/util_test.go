package integrationtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"google.golang.org/grpc"

	tmservicetypes "github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	govtype "github.com/cosmos/cosmos-sdk/x/gov/types"

	tmcrypto "github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/libs/bytes"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmhttp "github.com/tendermint/tendermint/rpc/client/http"
	tmtypes "github.com/tendermint/tendermint/types"
)

// copied from Tendermint type
type PVKey struct {
	Address tmtypes.Address  `json:"address"`
	PubKey  tmcrypto.PubKey  `json:"pub_key"`
	PrivKey tmcrypto.PrivKey `json:"priv_key"`
}

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

func applyVoteTallyingProposal(conn *grpc.ClientConn, proposalContent govtype.Content, proposerWallet *WalletInfo, voters []*WalletInfo) error {
	proposalId := uint64(0)

	{
		fmt.Println("Proposal apply")

		proposalMsg, err := govtype.NewMsgSubmitProposal(
			proposalContent,
			sdktypes.NewCoins(sdktypes.NewCoin("axpla", sdktypes.NewInt(10000000))),
			proposerWallet.ByteAddress,
		)

		if err != nil {
			return err
		}

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.Coin{
			Denom:  "axpla",
			Amount: feeAmt.Ceil().RoundInt(),
		}

		txhash, err := proposerWallet.SendTx(ChainID, proposalMsg, fee, xplaProposalGasLimit, false)
		if txhash != "" && err == nil {
			fmt.Println("Tx sent:", txhash)
		} else {
			return err
		}

		err = txCheck(txhash)
		if err == nil {
			fmt.Println("Tx applied", txhash)
		} else {
			return err
		}

		queryClient := txtypes.NewServiceClient(conn)
		resp, err := queryClient.GetTx(context.Background(), &txtypes.GetTxRequest{
			Hash: txhash,
		})

		if err != nil {
			return err
		}

	PROPOSAL_RAISED:
		for _, val := range resp.TxResponse.Events {
			for _, attr := range val.Attributes {
				if string(attr.Key) == "proposal_id" {
					proposalId, _ = strconv.ParseUint(string(attr.Value), 10, 64)
					break PROPOSAL_RAISED
				}
			}
		}

		fmt.Println("Proposal is applied as ID", proposalId)
	}

	{
		fmt.Println("Voting...")

		wg := sync.WaitGroup{}

		errChan := make(chan error)
		successChan := make(chan bool)

		for _, addr := range voters {
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
				if txhash != "" && err == nil {
					fmt.Println(addr.StringAddress, "voted to the proposal", proposalId, "as tx", txhash, "err:", err)
				} else {
					errChan <- err
					return
				}

				err = txCheck(txhash)
				if err == nil {
					fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
				} else {
					errChan <- err
					return
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
				return chanErr
			case <-successChan:
				break VOTE
			}
		}
	}

	fmt.Println("Waiting 25sec for the proposal passing...")
	time.Sleep(time.Second * 25)
	fmt.Println("Proposal tallied!")

	return nil
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

func getValidatorListOfLatestBlock(conn *grpc.ClientConn) ([]bytes.HexBytes, error) {
	client := tmservicetypes.NewServiceClient(conn)
	resp, err := client.GetLatestBlock(context.Background(), &tmservicetypes.GetLatestBlockRequest{})
	if err != nil {
		return nil, err
	}

	latestBlockHeight := resp.Block.GetHeader().Height
	fmt.Println("Height:", latestBlockHeight)

	tmclient, err := tmhttp.New("tcp://127.0.0.1:36657", "/websocket")
	if err != nil {
		return nil, err
	}

	blkresp, err := tmclient.Block(context.Background(), &latestBlockHeight)
	if err != nil {
		return nil, err
	}

	ret := []bytes.HexBytes{}
	for _, unitSign := range blkresp.Block.LastCommit.Signatures {
		ret = append(ret, unitSign.ValidatorAddress)
	}

	return ret, nil
}

func checkValidatorVoted(conn *grpc.ClientConn, validatorAddress bytes.HexBytes) (bool, error) {
	addrList, err := getValidatorListOfLatestBlock(conn)
	if err != nil {
		return false, err
	}

	fmt.Println("Given val address:", validatorAddress.String())

	for _, unitVal := range addrList {
		fmt.Println(unitVal.String())
		if validatorAddress.String() == unitVal.String() {
			return true, nil
		}
	}

	return false, nil
}
