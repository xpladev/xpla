package ante_test

import (
	sdkmath "cosmossdk.io/math"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/testutil/testdata"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	ibcclienttypes "github.com/cosmos/ibc-go/v10/modules/core/02-client/types"
	ibcchanneltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	feemarkettypes "github.com/cosmos/evm/x/feemarket/types"

	"github.com/xpladev/xpla/ante"
)

func (s *IntegrationTestSuite) TestMinGasPriceDecorator() {
	s.txBuilder = s.clientCtx.TxConfig.NewTxBuilder()

	s.app.FeeMarketKeeper.SetParams(s.ctx, feemarkettypes.NewParams(true, 8, 2, sdkmath.LegacyZeroDec(), 0, sdkmath.LegacyNewDec(200), sdkmath.LegacyMustNewDecFromStr("1.5")))

	mpd := ante.NewMinGasPriceDecorator(
		s.app.FeeMarketKeeper,
		s.app.EvmKeeper,
		[]string{
			sdk.MsgTypeURL(&ibcchanneltypes.MsgRecvPacket{}),
			sdk.MsgTypeURL(&ibcchanneltypes.MsgAcknowledgement{}),
			sdk.MsgTypeURL(&ibcclienttypes.MsgUpdateClient{}),
		})
	antehandler := sdk.ChainAnteDecorators(mpd)
	priv1, _, addr1 := testdata.KeyTestPubAddr()

	msg := testdata.NewTestMsg(addr1)
	feeAmount := testdata.NewTestFeeAmount()
	gasLimit := testdata.NewTestGasLimit()
	s.Require().NoError(s.txBuilder.SetMsgs(msg))
	s.txBuilder.SetFeeAmount(feeAmount)
	s.txBuilder.SetGasLimit(gasLimit)

	privs, accNums, accSeqs := []cryptotypes.PrivKey{priv1}, []uint64{0}, []uint64{0}
	tx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	s.Require().NoError(err)

	// Set high gas price so standard test fee fails
	feeAmt := sdk.NewDecCoinFromDec("uatom", sdkmath.LegacyNewDec(200).Quo(sdkmath.LegacyNewDec(100000)))
	minGasPrice := []sdk.DecCoin{feeAmt}
	s.ctx = s.ctx.WithMinGasPrices(minGasPrice).WithIsCheckTx(true)

	// antehandler errors with insufficient fees
	_, err = antehandler(s.ctx, tx, false)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFee)

	// simulations bypass minimum fee checks
	_, err = antehandler(s.ctx, tx, true)
	s.Require().NoError(err)

	s.ctx = s.ctx.WithIsCheckTx(false)

	// antehandler errors with insufficient fees during DeliverTx
	_, err = antehandler(s.ctx, tx, false)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFee)

	// ensure no fees for certain IBC msgs
	s.ctx = s.ctx.WithIsCheckTx(true)
	s.Require().NoError(s.txBuilder.SetMsgs(
		ibcchanneltypes.NewMsgRecvPacket(ibcchanneltypes.Packet{}, nil, ibcclienttypes.Height{}, ""),
	))

	oracleTx, err := s.CreateTestTx(privs, accNums, accSeqs, s.ctx.ChainID())
	_, err = antehandler(s.ctx, oracleTx, false)
	s.Require().NoError(err, "expected min fee bypass for IBC messages")

	s.ctx = s.ctx.WithIsCheckTx(false)

	// local minimum fee bypass configuration must not affect DeliverTx
	_, err = antehandler(s.ctx, oracleTx, false)
	s.Require().ErrorIs(err, sdkerrors.ErrInsufficientFee)
}
