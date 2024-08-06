package v1_6

import (
	"math/big"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	evmtypes "github.com/xpladev/ethermint/x/evm/types"
	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
	cdc codec.BinaryCodec,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		data, err := hexutil.Decode(thirdwebProxy)
		if err != nil {
			return nil, errors.Wrap(err, "failed to decode ethereum tx hex bytes")
		}

		msg := &evmtypes.MsgEthereumTx{}
		if err := msg.UnmarshalBinary(data); err != nil {
			return nil, err
		}

		if err := msg.ValidateBasic(); err != nil {
			return nil, err
		}

		res, err := keepers.EvmKeeper.EthereumTx(ctx, msg)
		if err != nil {
			return nil, err
		}

		if res.Failed() {
			return nil, errors.ErrPanic.Wrap(res.VmError)
		}

		// Gas refunded rollback without use
		tx := msg.AsTransaction()

		signer := ethtypes.NewLondonSigner(keepers.EvmKeeper.ChainID())
		coreMsg, err := msg.AsMessage(signer, keepers.FeeMarketKeeper.CalculateBaseFee(ctx))
		if err != nil {
			return nil, err
		}

		refundedGas := msg.GetGas() - res.GasUsed
		refundAmount := new(big.Int).Mul(new(big.Int).SetUint64(refundedGas), tx.GasPrice())
		evmDenom := keepers.EvmKeeper.GetParams(ctx).EvmDenom

		err = keepers.BankKeeper.SendCoinsFromAccountToModule(ctx, coreMsg.From().Bytes(), authtypes.FeeCollectorName, sdk.NewCoins(sdk.NewCoin(evmDenom, sdkmath.NewIntFromBigInt(refundAmount))))
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
