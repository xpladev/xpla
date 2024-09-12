package v1_7

import (
	"math/big"

	"cosmossdk.io/errors"
	sdkmath "cosmossdk.io/math"
	upgradetypes "cosmossdk.io/x/upgrade/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
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

		// fund fee collector by upgrade fee supporter
		upgradeFeeSupporterAccout, err := sdk.AccAddressFromBech32(upgradeFeeSupporter)
		if err != nil {
			return nil, err
		}
		evmDenom := keepers.EvmKeeper.GetParams(ctx).EvmDenom
		borrowedCoins := sdk.NewCoin(evmDenom, sdk.DefaultPowerReduction)
		err = keepers.BankKeeper.SendCoinsFromAccountToModule(ctx, upgradeFeeSupporterAccout, authtypes.FeeCollectorName, sdk.NewCoins(borrowedCoins))
		if err != nil {
			return nil, err
		}

		// execute thiredweb proxy contract
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
		from, err := signer.Sender(tx)
		if err != nil {
			return nil, err
		}

		// sender -> rewardDistributeAccount
		refundedGas := msg.GetGas() - res.GasUsed
		refundAmount := new(big.Int).Mul(new(big.Int).SetUint64(refundedGas), tx.GasPrice())
		refundCoin := sdk.NewCoin(evmDenom, sdkmath.NewIntFromBigInt(refundAmount))
		err = keepers.BankKeeper.SendCoins(ctx, from.Bytes(), upgradeFeeSupporterAccout, sdk.NewCoins(refundCoin))
		if err != nil {
			return nil, err
		}

		// feeCollector -> rewardDistributeAccount
		err = keepers.BankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, upgradeFeeSupporterAccout, sdk.NewCoins(borrowedCoins.Sub(refundCoin)))
		if err != nil {
			return nil, err
		}

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
