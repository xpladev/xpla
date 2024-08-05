package v1_6

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"

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

		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
