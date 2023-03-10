package keeper

import (
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	authKeeper     types.AccountKeeper
	bankKeeper     types.BankKeeper
	distKeeper     types.DistributionKeeper
	contractKeeper wasmTypes.ContractOpsKeeper
	viewKeeper     wasmTypes.ViewKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	ak types.AccountKeeper, bk types.BankKeeper, dk types.DistributionKeeper, ck wasmTypes.ContractOpsKeeper, vk wasmTypes.ViewKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:       key,
		cdc:            cdc,
		paramSpace:     paramSpace,
		authKeeper:     ak,
		bankKeeper:     bk,
		distKeeper:     dk,
		contractKeeper: ck,
		viewKeeper:     vk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetFeeInfoFromXATP(ctx sdk.Context, cw20 string) (sdk.Dec, error) {
	xatp, found := k.GetXatp(ctx, cw20)

	if !found {
		return sdk.Dec{}, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "not found xatp")
	}

	pool, err := k.Pool(ctx, xatp.Pair)
	if err != nil {
		return sdk.Dec{}, sdkerrors.Wrap(err, "fail to query pool")
	}

	nativeTokenAmout, tokenAmount, err := pool.Amount()
	if err != nil {
		return sdk.Dec{}, sdkerrors.Wrap(err, "invalid pool amount")
	}

	nativeTokenAmountDec := sdk.NewDecFromIntWithPrec(nativeTokenAmout, xplatypes.DefaultDenomPrecision)
	tokenAmountDec := sdk.NewDecFromIntWithPrec(tokenAmount, int64(xatp.Decimals))

	return tokenAmountDec.Quo(nativeTokenAmountDec), nil
}

func (k Keeper) PayXATP(ctx sdk.Context, deductFeesFrom sdk.AccAddress, denom string, amount string) error {
	xatpPayer := k.GetXatpPayerAccount()

	xatp, found := k.GetXatp(ctx, denom)
	if !found {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "denom")
	}

	err := k.TransferCw20(ctx, deductFeesFrom, xatp.Token, amount, xatpPayer.String())
	if err != nil {
		return err
	}

	return nil
}
