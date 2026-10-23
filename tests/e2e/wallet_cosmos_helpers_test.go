package e2e

import (
	"context"
	"net"
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	cryptocodec "github.com/cosmos/evm/crypto/codec"
	"github.com/cosmos/evm/crypto/ethsecp256k1"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	xplatypes "github.com/xpladev/xpla/types"
)

type walletTxService struct {
	txtypes.UnimplementedServiceServer
	simulated    []byte
	broadcast    []byte
	mode         txtypes.BroadcastMode
	code         uint32
	broadcastErr error
}

func (s *walletTxService) Simulate(_ context.Context, req *txtypes.SimulateRequest) (*txtypes.SimulateResponse, error) {
	s.simulated = req.TxBytes
	return &txtypes.SimulateResponse{GasInfo: &sdk.GasInfo{GasUsed: 100_000}}, nil
}

func (s *walletTxService) BroadcastTx(_ context.Context, req *txtypes.BroadcastTxRequest) (*txtypes.BroadcastTxResponse, error) {
	s.broadcast = req.TxBytes
	s.mode = req.Mode
	if s.broadcastErr != nil {
		return nil, s.broadcastErr
	}
	return &txtypes.BroadcastTxResponse{TxResponse: &sdk.TxResponse{Code: s.code, TxHash: "test-tx-hash", RawLog: "packet already processed"}}, nil
}

func TestWalletInfoSendTx(t *testing.T) {
	t.Setenv("GOLANG_TESTING", "false")
	previousDesc, previousAdjustment := desc, gasAdjustment
	t.Cleanup(func() { desc, gasAdjustment = previousDesc, previousAdjustment })
	gasAdjustment = sdkmath.LegacyNewDecWithPrec(15, 1)

	for _, tc := range []struct {
		name         string
		code         uint32
		broadcastErr error
		wantError    string
	}{
		{name: "success advances sequence"},
		{name: "rejected replay preserves sequence", code: 22, wantError: "Tx failed with code 22"},
		{name: "transport failure preserves sequence", broadcastErr: status.Error(codes.Unavailable, "broadcast unavailable"), wantError: "broadcast unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &walletTxService{code: tc.code, broadcastErr: tc.broadcastErr}
			listener := bufconn.Listen(1024 * 1024)
			server := grpc.NewServer()
			txtypes.RegisterServiceServer(server, service)
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Stop(); _ = listener.Close() })
			conn, err := grpc.NewClient("passthrough:///wallet-test", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return listener.Dial() }))
			require.NoError(t, err)
			t.Cleanup(func() { _ = conn.Close() })
			desc = &ServiceDesc{ServiceConn: conn}

			encCfg := moduletestutil.MakeTestEncodingConfig()
			banktypes.RegisterInterfaces(encCfg.InterfaceRegistry)
			cryptocodec.RegisterInterfaces(encCfg.InterfaceRegistry)
			key, err := ethsecp256k1.GenerateKey()
			require.NoError(t, err)
			wallet := &WalletInfo{ChainId: ChainID, PrivKey: key, PubKey: key.PubKey(), AccountNumber: 9, Sequence: 7, EncCfg: encCfg}
			address := sdk.AccAddress(key.PubKey().Address()).String()
			msg := &banktypes.MsgSend{FromAddress: address, ToAddress: address, Amount: sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 1))}

			hash, err := wallet.SendTx(ChainID, msg, false)
			if tc.wantError != "" {
				require.ErrorContains(t, err, tc.wantError)
				require.Empty(t, hash)
				require.Equal(t, uint64(7), wallet.Sequence)
			} else {
				require.NoError(t, err)
				require.Equal(t, "test-tx-hash", hash)
				require.Equal(t, uint64(8), wallet.Sequence)
			}
			require.Equal(t, txtypes.BroadcastMode_BROADCAST_MODE_SYNC, service.mode)
			require.NotEqual(t, service.simulated, service.broadcast)

			for _, encoded := range [][]byte{service.simulated, service.broadcast} {
				decoded, err := encCfg.TxConfig.TxDecoder()(encoded)
				require.NoError(t, err)
				require.Equal(t, []sdk.Msg{msg}, decoded.GetMsgs())
				var raw txtypes.TxRaw
				require.NoError(t, raw.Unmarshal(encoded))
				var info txtypes.AuthInfo
				require.NoError(t, info.Unmarshal(raw.AuthInfoBytes))
				require.Len(t, info.SignerInfos, 1)
				require.Equal(t, uint64(7), info.SignerInfos[0].Sequence)
				require.Equal(t, signing.SignMode_SIGN_MODE_DIRECT, info.SignerInfos[0].ModeInfo.GetSingle().Mode)
				doc := txtypes.SignDoc{BodyBytes: raw.BodyBytes, AuthInfoBytes: raw.AuthInfoBytes, ChainId: ChainID, AccountNumber: 9}
				signBytes, err := doc.Marshal()
				require.NoError(t, err)
				require.Len(t, raw.Signatures, 1)
				require.True(t, key.PubKey().VerifySignature(signBytes, raw.Signatures[0]), "encoded transaction must carry a valid DIRECT signature")
			}
			decoded, err := encCfg.TxConfig.TxDecoder()(service.broadcast)
			require.NoError(t, err)
			feeTx := decoded.(sdk.FeeTx)
			require.Equal(t, uint64(150_000), feeTx.GetGas())
			require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(xplatypes.DefaultDenom, 42_000_000_000_000_000)), feeTx.GetFee())
		})
	}
}
