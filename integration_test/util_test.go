package integrationtest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1type "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govv1beta1type "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	stakingtype "github.com/cosmos/cosmos-sdk/x/staking/types"

	tmcrypto "github.com/cometbft/cometbft/crypto"
	"github.com/cometbft/cometbft/libs/bytes"
	tmjson "github.com/cometbft/cometbft/libs/json"
	tmhttp "github.com/cometbft/cometbft/rpc/client/http"
	tmtypes "github.com/cometbft/cometbft/types"

	xplatypes "github.com/xpladev/xpla/types"
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
	volunteerValidatorWallet1, volunteerValidatorWallet2, volunteerValidatorWallet3 *WalletInfo,
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

	volunteerValidator1, err := os.ReadFile(filepath.Join(".", "test_keys", "volunteer_validator1.mnemonics"))
	if err != nil {
		panic(err)
	}

	volunteerValidatorWallet1, err = NewWalletInfo(string(volunteerValidator1))
	if err != nil {
		panic(err)
	}

	volunteerValidator2, err := os.ReadFile(filepath.Join(".", "test_keys", "volunteer_validator2.mnemonics"))
	if err != nil {
		panic(err)
	}

	volunteerValidatorWallet2, err = NewWalletInfo(string(volunteerValidator2))
	if err != nil {
		panic(err)
	}

	volunteerValidator3, err := os.ReadFile(filepath.Join(".", "test_keys", "volunteer_validator3.mnemonics"))
	if err != nil {
		panic(err)
	}

	volunteerValidatorWallet3, err = NewWalletInfo(string(volunteerValidator3))
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

		time.Sleep(time.Second / 5)
	}

	return err
}

func applyVoteTallyingProposal(conn *grpc.ClientConn, proposalMsgs []sdktypes.Msg, proposalContent govv1beta1type.Content, proposerWallet *WalletInfo, voters []*WalletInfo) error {
	proposalId := uint64(0)

	{
		fmt.Println("Proposal apply")

		var msg sdktypes.Msg
		var err error
		if len(proposalMsgs) > 0 {
			msg, err = govv1type.NewMsgSubmitProposal(proposalMsgs, sdktypes.NewCoins(sdktypes.NewCoin(xplatypes.DefaultDenom, sdktypes.NewInt(10000000))), proposerWallet.ByteAddress.String(), "", proposalContent.GetTitle(), proposalContent.GetDescription())
			if err != nil {
				return err
			}

		} else {
			msg, err = govv1beta1type.NewMsgSubmitProposal(
				proposalContent,
				sdktypes.NewCoins(sdktypes.NewCoin(xplatypes.DefaultDenom, sdktypes.NewInt(10000000))),
				proposerWallet.ByteAddress,
			)
			if err != nil {
				return err
			}
		}

		if err != nil {
			return err
		}

		feeAmt := sdktypes.NewDec(xplaProposalGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
		fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

		txhash, err := proposerWallet.SendTx(ChainID, msg, fee, xplaProposalGasLimit, false)
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

		var eg errgroup.Group

		for _, addr := range voters {
			addr := addr

			eg.Go(func() error {
				voteMsg := govv1beta1type.NewMsgVote(addr.ByteAddress, proposalId, govv1beta1type.OptionYes)
				feeAmt := sdktypes.NewDec(xplaGeneralGasLimit).Mul(sdktypes.MustNewDecFromStr(xplaGasPrice))
				fee := sdktypes.NewCoin(xplatypes.DefaultDenom, feeAmt.Ceil().RoundInt())

				txhash, err := addr.SendTx(ChainID, voteMsg, fee, xplaGeneralGasLimit, false)
				if txhash != "" && err == nil {
					fmt.Println(addr.StringAddress, "voted to the proposal", proposalId, "as tx", txhash, "err:", err)
				} else {
					return err
				}

				err = txCheck(txhash)
				if err == nil {
					fmt.Println(addr.StringAddress, "vote tx applied", txhash, "err:", err)
				} else {
					return err
				}

				return nil
			})
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	fmt.Println("Waiting 4 blocktime for the proposal passing...")
	time.Sleep(time.Second*blocktime*proposalBlocks + 1)
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
	client, err := tmhttp.New("tcp://127.0.0.1:36657", "/websocket")
	if err != nil {
		return nil, err
	}

	// nil: the latest block
	blkresp, err := client.Block(context.Background(), nil)
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

func getValidatorBondingState(conn *grpc.ClientConn, addr sdktypes.ValAddress) (stakingtype.BondStatus, error) {
	client := stakingtype.NewQueryClient(conn)

	resp, err := client.Validator(
		context.Background(),
		&stakingtype.QueryValidatorRequest{ValidatorAddr: addr.String()},
	)

	if err != nil {
		return stakingtype.Unspecified, err
	}

	return resp.Validator.Status, nil
}

func makeUpdateParamMaxValidators(conn *grpc.ClientConn, maxValidators uint32) (sdktypes.Msg, error) {
	stakingQueryClient := stakingtype.NewQueryClient(conn)
	resStakingParams, err := stakingQueryClient.Params(context.Background(), &stakingtype.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	authQueryClient := authtypes.NewQueryClient(conn)
	resModuleAccount, err := authQueryClient.ModuleAccountByName(context.Background(), &authtypes.QueryModuleAccountByNameRequest{Name: govtypes.ModuleName})
	if err != nil {
		return nil, err
	}

	// change MaxValidators
	resStakingParams.Params.MaxValidators = maxValidators

	var moduleAccount authtypes.AccountI
	err = marshaler.UnpackAny(resModuleAccount.Account, &moduleAccount)
	if err != nil {
		return nil, err
	}

	msgUpdateParams := stakingtype.MsgUpdateParams{
		Authority: moduleAccount.GetAddress().String(),
		Params:    resStakingParams.Params,
	}

	m, err := marshaler.MarshalInterface(&msgUpdateParams)
	if err != nil {
		return nil, err
	}

	var msg sdktypes.Msg
	err = marshaler.UnmarshalInterface(m, &msg)
	if err != nil {
		return nil, err
	}

	return msg, nil
}
