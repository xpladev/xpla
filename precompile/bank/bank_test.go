package bank

import (
	"context"
	"errors"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/xpladev/xpla/precompile/util"
)

type recordingBankKeeper struct {
	BankKeeper

	balanceCalls int
	supplyCalls  int
}

func (k *recordingBankKeeper) GetBalance(context.Context, sdk.AccAddress, string) sdk.Coin {
	k.balanceCalls++
	return sdk.Coin{}
}

func (k *recordingBankKeeper) GetSupply(context.Context, string) sdk.Coin {
	k.supplyCalls++
	return sdk.Coin{}
}

func TestViewMethodsRejectMalformedReservedDenomBeforeKeeperCall(t *testing.T) {
	t.Run("balance", func(t *testing.T) {
		keeper := &recordingBankKeeper{}
		precompile := PrecompiledBank{bk: keeper}
		method := ABI.Methods[string(Balance)]

		_, err := precompile.balance(sdk.Context{}, &method, []interface{}{common.Address{}, "xerc20:invalid"})

		require.Error(t, err)
		require.Zero(t, keeper.balanceCalls)
	})

	t.Run("supply", func(t *testing.T) {
		keeper := &recordingBankKeeper{}
		precompile := PrecompiledBank{bk: keeper}
		method := ABI.Methods[string(Supply)]

		_, err := precompile.supplyOf(sdk.Context{}, &method, []interface{}{"xcw20:invalid"})

		require.Error(t, err)
		require.Zero(t, keeper.supplyCalls)
	})
}

var errSendCoinsReached = errors.New("sentinel: send coins reached")

type sendBankKeeper struct {
	BankKeeper

	sendEnabledErr error
	blocked        bool

	isSendEnabledCalls int
	blockedAddrCalls   int
	sendCalls          int

	enabledCoins sdk.Coins
	blockedAddr  sdk.AccAddress
	sentFrom     sdk.AccAddress
	sentTo       sdk.AccAddress
	sentCoins    sdk.Coins
}

func (k *sendBankKeeper) IsSendEnabledCoins(_ context.Context, coins ...sdk.Coin) error {
	k.isSendEnabledCalls++
	k.enabledCoins = sdk.NewCoins(coins...)
	return k.sendEnabledErr
}

func (k *sendBankKeeper) BlockedAddr(addr sdk.AccAddress) bool {
	k.blockedAddrCalls++
	k.blockedAddr = addr
	return k.blocked
}

func (k *sendBankKeeper) SendCoins(_ context.Context, from, to sdk.AccAddress, coins sdk.Coins) error {
	k.sendCalls++
	k.sentFrom = from
	k.sentTo = to
	k.sentCoins = coins
	return errSendCoinsReached
}

func TestSendAppliesMsgSendPolicyBeforeSendCoins(t *testing.T) {
	from := common.HexToAddress("0x1111111111111111111111111111111111111111")
	to := common.HexToAddress("0x2222222222222222222222222222222222222222")
	method := ABI.Methods[string(Send)]

	call := func(t *testing.T, keeper *sendBankKeeper, coins []util.Coin) error {
		t.Helper()
		precompile := PrecompiledBank{bk: keeper}
		_, err := precompile.send(sdk.Context{}, nil, from, &method, []interface{}{from, to, coins})
		return err
	}

	positive := func(denom string) []util.Coin {
		return []util.Coin{{Denom: denom, Amount: big.NewInt(10)}}
	}

	t.Run("empty amount", func(t *testing.T) {
		keeper := &sendBankKeeper{
			sendEnabledErr: banktypes.ErrSendDisabled,
			blocked:        true,
		}

		err := call(t, keeper, []util.Coin{})

		require.ErrorIs(t, err, sdkerrors.ErrInvalidCoins)
		require.Zero(t, keeper.isSendEnabledCalls)
		require.Zero(t, keeper.blockedAddrCalls)
		require.Zero(t, keeper.sendCalls)
	})

	t.Run("send disabled", func(t *testing.T) {
		keeper := &sendBankKeeper{sendEnabledErr: banktypes.ErrSendDisabled}

		err := call(t, keeper, positive("axpla"))

		require.ErrorIs(t, err, banktypes.ErrSendDisabled)
		require.Equal(t, 1, keeper.isSendEnabledCalls)
		require.Zero(t, keeper.blockedAddrCalls)
		require.Zero(t, keeper.sendCalls)
	})

	xerc20Denom := "xerc20:0xA2dC463DD29be4C8a28dB0C09D89b0AA89Fc9546"
	for _, denom := range []string{"axpla", xerc20Denom} {
		t.Run("blocked recipient "+denom, func(t *testing.T) {
			keeper := &sendBankKeeper{blocked: true}

			err := call(t, keeper, positive(denom))

			require.ErrorIs(t, err, sdkerrors.ErrUnauthorized)
			require.ErrorContains(t, err, sdk.AccAddress(to.Bytes()).String()+" is not allowed to receive funds")
			require.Equal(t, 1, keeper.isSendEnabledCalls)
			if denom == xerc20Denom {
				require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin(denom, 10)), keeper.enabledCoins)
			}
			require.Equal(t, 1, keeper.blockedAddrCalls)
			require.Equal(t, sdk.AccAddress(to.Bytes()), keeper.blockedAddr)
			require.Zero(t, keeper.sendCalls)
		})
	}

	t.Run("passes policy and reaches send", func(t *testing.T) {
		keeper := &sendBankKeeper{}

		err := call(t, keeper, positive("axpla"))

		require.ErrorIs(t, err, errSendCoinsReached)
		require.Equal(t, 1, keeper.sendCalls)
		require.Equal(t, sdk.AccAddress(from.Bytes()), keeper.sentFrom)
		require.Equal(t, sdk.AccAddress(to.Bytes()), keeper.sentTo)
		require.Equal(t, sdk.NewCoins(sdk.NewInt64Coin("axpla", 10)), keeper.sentCoins)
	})
}
