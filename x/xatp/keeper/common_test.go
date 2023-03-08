package keeper_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/std"
	"github.com/cosmos/cosmos-sdk/store"
	"github.com/cosmos/cosmos-sdk/types/module"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	"github.com/cosmos/cosmos-sdk/x/auth/tx"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	params "github.com/cosmos/cosmos-sdk/x/params"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/libs/log"
	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
	dbm "github.com/tendermint/tm-db"
	"github.com/xpladev/xpla/x/xatp/keeper"
	"github.com/xpladev/xpla/x/xatp/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramskeeper "github.com/cosmos/cosmos-sdk/x/params/keeper"
)

func init() {
	sdk.DefaultPowerReduction = sdk.NewIntFromBigInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil))
}

// ModuleBasics nolint
var ModuleBasics = module.NewBasicManager(
	params.AppModuleBasic{},
)

// MakeEncodingConfig nolint
func MakeEncodingConfig(_ *testing.T) simparams.EncodingConfig {
	amino := codec.NewLegacyAmino()
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	marshaler := codec.NewProtoCodec(interfaceRegistry)
	txCfg := tx.NewTxConfig(marshaler, tx.DefaultSignModes)

	std.RegisterInterfaces(interfaceRegistry)
	std.RegisterLegacyAminoCodec(amino)

	ModuleBasics.RegisterLegacyAminoCodec(amino)
	ModuleBasics.RegisterInterfaces(interfaceRegistry)
	return simparams.EncodingConfig{
		InterfaceRegistry: interfaceRegistry,
		Marshaler:         marshaler,
		TxConfig:          txCfg,
		Amino:             amino,
	}
}

// createTestInput Returns a simapp
func createTestInput(t *testing.T) (xatpKeeper keeper.Keeper, ctx sdk.Context) {
	keyAcc := sdk.NewKVStoreKey(authtypes.StoreKey)
	keyParams := sdk.NewKVStoreKey(paramstypes.StoreKey)
	keyXatp := sdk.NewKVStoreKey(types.StoreKey)
	tKeyParams := sdk.NewTransientStoreKey(paramstypes.TStoreKey)

	db := dbm.NewMemDB()
	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(keyAcc, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyXatp, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(keyParams, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(tKeyParams, sdk.StoreTypeTransient, db)

	maccPerms := map[string][]string{
		authtypes.FeeCollectorName: nil,
		types.ModuleName:           nil,
	}

	require.NoError(t, ms.LoadLatestVersion())

	encodingConfig := MakeEncodingConfig(t)
	appCodec, legacyAmino := encodingConfig.Marshaler, encodingConfig.Amino

	paramsKeeper := paramskeeper.NewKeeper(appCodec, legacyAmino, keyParams, tKeyParams)
	accountKeeper := authkeeper.NewAccountKeeper(appCodec, keyAcc, paramsKeeper.Subspace(authtypes.ModuleName), authtypes.ProtoBaseAccount, maccPerms)
	_, wasmKeeper := wasmkeeper.CreateTestInput(t, false, "iterator,staking,stargate")
	xatpKeeper = keeper.NewKeeper(appCodec, keyXatp, paramsKeeper.Subspace(types.ModuleName), accountKeeper, wasmKeeper.ContractKeeper, wasmKeeper.WasmKeeper)

	ctx = sdk.NewContext(ms, tmproto.Header{Time: time.Now().UTC()}, false, log.NewNopLogger())
	xatpKeeper.SetParams(ctx, types.DefaultParams())

	return xatpKeeper, ctx
}
