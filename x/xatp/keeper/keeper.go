package keeper

import (
	"encoding/json"
	"math/big"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

type WasmExecute struct {
	Cdc      codec.Codec
	StoreKey sdk.StoreKey
}

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	contractKeeper wasmTypes.ContractOpsKeeper
	viewKeeper     wasmTypes.ViewKeeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	ck wasmTypes.ContractOpsKeeper, vk wasmTypes.ViewKeeper,
) Keeper {
	// set KeyTable if it has not already been set
	if !paramSpace.HasKeyTable() {
		paramSpace = paramSpace.WithKeyTable(types.ParamKeyTable())
	}

	return Keeper{
		storeKey:       key,
		cdc:            cdc,
		paramSpace:     paramSpace,
		contractKeeper: ck,
		viewKeeper:     vk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k Keeper) GetFeeInfoFromXATP(ctx sdk.Context, cw20 string) (sdk.Dec, error, sdk.AccAddress) {

	var pair string
	var contract string
	cw20Denom, found := k.GetXatp(ctx, cw20)

	if !found {
		return sdk.Dec{}, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "Not found pair address"), nil
	}

	pair = cw20Denom.Pair
	contract = cw20Denom.Token

	PairContractAddr, err := sdk.AccAddressFromBech32(pair)

	bz, err := k.viewKeeper.QuerySmart(ctx, PairContractAddr, []byte(`{"pair":{}}`))

	type PairInfo struct {
		AssetDecimals []int `json:"asset_decimals"`
		AssetInfos    []struct {
			Token struct {
				ContractAddr string `json:"contract_addr"`
			} `json:"token"`
			NativeToken struct {
				Denom string `json:"denom"`
			} `json:"native_token"`
		} `json:"asset_infos"`
		ContractAddr   string `json:"contract_addr"`
		LiquidityToken string `json:"liquidity_token"`
	}

	pairInfo := PairInfo{}
	err = json.Unmarshal(bz, &pairInfo)
	if err != nil {
		return sdk.Dec{}, sdkerrors.Wrapf(sdkerrors.ErrJSONUnmarshal, "get pair info from XATP unmarshal error"), nil
	}

	var decimals [2]int

	for i, asset := range pairInfo.AssetInfos {

		if asset.NativeToken.Denom == xplatypes.DefaultDenom {
			decimals[0] = pairInfo.AssetDecimals[i]
			continue
		}

		if asset.Token.ContractAddr == contract {
			decimals[1] = pairInfo.AssetDecimals[i]
			continue
		}

	}

	bz, err = k.viewKeeper.QuerySmart(ctx, PairContractAddr, []byte(`{"pool":{}}`))

	type PoolInfo struct {
		Assets []struct {
			Amount string `json:"amount"`
			Info   struct {
				Token struct {
					ContractAddr string `json:"contract_addr"`
				} `json:"token"`
				NativeToken struct {
					Denom string `json:"denom"`
				} `json:"native_token"`
			} `json:"info"`
		} `json:"assets"`
		TotalShare string `json:"total_share"`
	}

	var poolInfo PoolInfo

	err = json.Unmarshal(bz, &poolInfo)
	if err != nil {
		return sdk.Dec{}, sdkerrors.Wrapf(sdkerrors.ErrJSONUnmarshal, "get pool info from XATP unmarshal error"), nil
	}

	var amounts [2]big.Int
	for i, asset := range poolInfo.Assets {

		if asset.Info.NativeToken.Denom == xplatypes.DefaultDenom {
			amounts[0].SetString(poolInfo.Assets[i].Amount, 10)
			continue
		}

		if asset.Info.Token.ContractAddr == contract {
			amounts[1].SetString(poolInfo.Assets[i].Amount, 10)
			continue
		}

	}

	gap := decimals[0] - decimals[1]

	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(gap)), nil)

	amounts[1] = *new(big.Int).Mul(&amounts[1], power)
	ratio := new(big.Int).Div(&amounts[1], &amounts[0])
	ratioDec, err := sdk.NewDecFromStr(ratio.String())
	contractAddr, err := sdk.AccAddressFromBech32(contract)

	return ratioDec, err, contractAddr
}

func (k Keeper) ExecuteContract(ctx sdk.Context, deductFeesFrom sdk.AccAddress, denom string, amount string) error {

	var xatpPayer string
	k.paramSpace.Get(ctx, types.ParamStoreKeyPayer, &xatpPayer)

	_, err, contractAddr := k.GetFeeInfoFromXATP(ctx, denom)
	if err != nil {
		return err
	}

	msg :=
		`
			{
				"transfer": {
					"recipient":  "` + xatpPayer + `",
					"amount": "` + amount + `"
				}
			}
		`

	_, err = k.contractKeeper.Execute(
		ctx,
		contractAddr,
		deductFeesFrom,
		wasmTypes.RawContractMessage(msg),
		sdk.Coins{},
	)
	if err != nil {
		return err
	}

	return nil
}
