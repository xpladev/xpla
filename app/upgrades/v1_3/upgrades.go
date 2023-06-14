package v1_3

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	icahosttypes "github.com/cosmos/ibc-go/v4/modules/apps/27-interchain-accounts/host/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	"github.com/xpladev/xpla/app/keepers"
)

func CreateUpgradeHandler(
	mm *module.Manager,
	configurator module.Configurator,
	keepers *keepers.AppKeepers,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, plan upgradetypes.Plan, fromVM module.VersionMap) (module.VersionMap, error) {
		// update ICA Host to add new messages available
		// enumerate all because it's easier to reason about
		newIcaHostParams := icahosttypes.Params{
			HostEnabled: true,
			AllowMessages: []string{
				sdk.MsgTypeURL(&ibctransfertypes.MsgTransfer{}), // added in v10

				sdk.MsgTypeURL(&banktypes.MsgSend{}),
				sdk.MsgTypeURL(&banktypes.MsgMultiSend{}), // this was missed last time
				sdk.MsgTypeURL(&stakingtypes.MsgDelegate{}),
				sdk.MsgTypeURL(&stakingtypes.MsgUndelegate{}), // added in v10
				sdk.MsgTypeURL(&stakingtypes.MsgBeginRedelegate{}),
				sdk.MsgTypeURL(&stakingtypes.MsgCreateValidator{}),
				sdk.MsgTypeURL(&stakingtypes.MsgEditValidator{}),
				sdk.MsgTypeURL(&distrtypes.MsgWithdrawDelegatorReward{}),
				sdk.MsgTypeURL(&distrtypes.MsgSetWithdrawAddress{}),
				sdk.MsgTypeURL(&distrtypes.MsgWithdrawValidatorCommission{}),
				sdk.MsgTypeURL(&distrtypes.MsgFundCommunityPool{}),
				sdk.MsgTypeURL(&govtypes.MsgVote{}),
				sdk.MsgTypeURL(&govtypes.MsgVoteWeighted{}), // added in v10
				sdk.MsgTypeURL(&authz.MsgExec{}),
				sdk.MsgTypeURL(&authz.MsgGrant{}),
				sdk.MsgTypeURL(&authz.MsgRevoke{}),
				// wasm msgs here
				// note we only support three atm (well four inc instantiate2)
				sdk.MsgTypeURL(&wasmtypes.MsgStoreCode{}),
				sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract{}),
				sdk.MsgTypeURL(&wasmtypes.MsgInstantiateContract2{}), // added in wasmd 0.29.0
				sdk.MsgTypeURL(&wasmtypes.MsgExecuteContract{}),
			},
		}
		keepers.ICAHostKeeper.SetParams(ctx, newIcaHostParams)

		// transfer module consensus version has been bumped to 2
		return mm.RunMigrations(ctx, configurator, fromVM)
	}
}
