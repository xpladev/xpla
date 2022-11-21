package keeper

import (
	"encoding/json"
	"math/big"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"
	xplatypes "github.com/xpladev/xpla/types"
	"github.com/xpladev/xpla/x/xatp/types"
)

type WasmExecute struct {
	Cdc      codec.Codec
	StoreKey sdk.StoreKey
}

type WasmExecuteInterface interface {
	ContractInstance(ctx sdk.Context, contractAddress sdk.AccAddress) (wasmTypes.ContractInfo, wasmTypes.CodeInfo, prefix.Store, error)
}

type Keeper struct {
	storeKey   sdk.StoreKey
	cdc        codec.BinaryCodec
	paramSpace paramtypes.Subspace

	contractKeeper wasmTypes.ContractOpsKeeper
	viewKeeper     wasmTypes.ViewKeeper
	wasmExecute    WasmExecuteInterface
	wasmKeeper     wasmkeeper.Keeper
}

func NewKeeper(
	cdc codec.BinaryCodec, key sdk.StoreKey, paramSpace paramtypes.Subspace,
	ck wasmTypes.ContractOpsKeeper, vk wasmTypes.ViewKeeper, we WasmExecuteInterface, wk wasmkeeper.Keeper,
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
		wasmExecute:    we,
		wasmKeeper:     wk,
	}
}

// Logger returns a module-specific logger.
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", "x/"+types.ModuleName)
}

func (k WasmExecute) ContractInstance(ctx sdk.Context, contractAddress sdk.AccAddress) (wasmTypes.ContractInfo, wasmTypes.CodeInfo, prefix.Store, error) {
	store := ctx.KVStore(k.StoreKey)

	contractBz := store.Get(wasmTypes.GetContractAddressKey(contractAddress))
	if contractBz == nil {
		return wasmTypes.ContractInfo{}, wasmTypes.CodeInfo{}, prefix.Store{}, sdkerrors.Wrap(wasmTypes.ErrNotFound, "contract")
	}
	var contractInfo wasmTypes.ContractInfo
	k.Cdc.MustUnmarshal(contractBz, &contractInfo)

	codeInfoBz := store.Get(wasmTypes.GetCodeKey(contractInfo.CodeID))
	if codeInfoBz == nil {
		return contractInfo, wasmTypes.CodeInfo{}, prefix.Store{}, sdkerrors.Wrap(wasmTypes.ErrNotFound, "code info")
	}
	var codeInfo wasmTypes.CodeInfo
	k.Cdc.MustUnmarshal(codeInfoBz, &codeInfo)
	prefixStoreKey := wasmTypes.GetContractStorePrefix(contractAddress)
	prefixStore := prefix.NewStore(ctx.KVStore(k.StoreKey), prefixStoreKey)
	return contractInfo, codeInfo, prefixStore, nil
}

func (k Keeper) GetFeeInfoFromXATP(ctx sdk.Context, cw20 string) (sdk.Dec, error, sdk.AccAddress) {

	var pair string
	var contract string
	cw20Denoms := k.GetXATPs(ctx)

	for i := 0; i < len(cw20Denoms); i++ {
		if cw20Denoms[i].Denom == cw20 {
			pair = cw20Denoms[i].Pair
			contract = cw20Denoms[i].Contract
			break
		}
	}

	if len(pair) == 0 {
		return sdk.Dec{}, sdkerrors.Wrapf(sdkerrors.ErrNotFound, "Not found pair address"), nil
	}

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
	k.paramSpace.Get(ctx, types.ParamStoreKeyXATPPayer, &xatpPayer)

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

func (k Keeper) GetExecuteCost(ctx sdk.Context, denom string) (uint64, error) {

	_, err, contractAddr := k.GetFeeInfoFromXATP(ctx, denom)
	if err != nil {
		return 0, err
	}

	contractInfo, _, _, err := k.wasmExecute.ContractInstance(ctx, contractAddr)

	if err != nil {
		return 0, err
	}

	var xatpPayer string
	k.paramSpace.Get(ctx, types.ParamStoreKeyXATPPayer, &xatpPayer)

	msg :=
		`
		{
			"transfer": {
				"recipient":  "` + xatpPayer + `",
				"amount": "1000000"
			}
		}
	`
	executeCosts := wasmkeeper.NewDefaultWasmGasRegister().InstantiateContractCosts(
		k.wasmKeeper.IsPinnedCode(ctx, contractInfo.CodeID),
		len(wasmTypes.RawContractMessage(msg)),
	)

	return executeCosts, nil
}
