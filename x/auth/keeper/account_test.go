package keeper

import (
	"testing"

	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	ctestutil "github.com/cosmos/cosmos-sdk/codec/testutil"
	"github.com/cosmos/cosmos-sdk/runtime"
	"github.com/cosmos/cosmos-sdk/testutil"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authcodec "github.com/cosmos/cosmos-sdk/x/auth/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/assert"
)

func TestGetSliceAddress(t *testing.T) {

	storeKey := storetypes.NewKVStoreKey(authtypes.ModuleName)
	tKey := storetypes.NewTransientStoreKey("transient_test")
	ctx := testutil.DefaultContext(storeKey, tKey)

	interfaceRegistry := ctestutil.CodecOptions{}.NewInterfaceRegistry()
	authtypes.RegisterInterfaces(interfaceRegistry)
	cdc := codec.NewProtoCodec(interfaceRegistry)
	accountKeeper := NewAccountKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		authtypes.ProtoBaseAccount,
		map[string][]string{},
		authcodec.NewBech32Codec(sdk.Bech32MainPrefix),
		sdk.Bech32MainPrefix,
		authtypes.NewModuleAddress(govtypes.ModuleName).String(),
	)

	original := sdk.MustAccAddressFromBech32("cosmos1qg5ega6dykkxc307y25pecuufrjkxkaggkkxh7nad0vhyhtuhw3s6ufdm4")
	valid := sdk.MustAccAddressFromBech32("cosmos1rn3ecj89vdd6s3dvd0a8667ewfwhewarkkd5wr")
	invalid := sdk.MustAccAddressFromBech32("cosmos1qg5ega6dykkxc307y25pecuufrjkxkags0q9gu")

	accountKeeper.SetAccount(ctx, authtypes.NewBaseAccount(original, nil, 0, 0))

	assert.True(t, accountKeeper.HasAccount(ctx, original))
	assert.True(t, accountKeeper.HasAccount(ctx, valid))
	assert.False(t, accountKeeper.HasAccount(ctx, invalid))

}
