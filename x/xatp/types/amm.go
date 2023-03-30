package types

import (
	"errors"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

type AssetCWToken struct {
	ContractAddr string `json:"contract_addr,omitempty"`
}
type AssetNativeToken struct {
	Denom string `json:"denom,omitempty"`
}

type AssetInfo struct {
	Token       *AssetCWToken     `json:"token,omitempty"`
	NativeToken *AssetNativeToken `json:"native_token,omitempty"`
}

type Pair struct {
	AssetDecimals  []int       `json:"asset_decimals,omitempty"`
	AssetInfos     []AssetInfo `json:"asset_infos,omitempty"`
	ContractAddr   string      `json:"contract_addr,omitempty"`
	LiquidityToken string      `json:"liquidity_token,omitempty"`
}

func (p Pair) Xatp() (
	token *AssetCWToken,
	tokenDecimal int, err error) {
	if p.AssetInfos[0].NativeToken != nil && p.AssetInfos[1].Token != nil {
		return p.AssetInfos[1].Token, p.AssetDecimals[1], nil
	} else if p.AssetInfos[0].Token != nil && p.AssetInfos[1].NativeToken != nil {
		return p.AssetInfos[0].Token, p.AssetDecimals[0], nil
	}

	return nil, -1, errors.New("can't be xatp")
}

type Token struct {
	Name        string `json:"name,omitempty"`
	Symbol      string `json:"symbol,omitempty"`
	Decimals    int    `json:"decimals,omitempty"`
	TotalSupply string `json:"total_supply,omitempty"`
}

type Pool struct {
	Assets []struct {
		Info   AssetInfo `json:"info,omitempty"`
		Amount string    `json:"amount,omitempty"`
	} `json:"assets,omitempty"`
	TotalShare string `json:"total_share,omitempty"`
}

func (p Pool) Amount() (nativeTokenAmount sdk.Int, tokenAmount sdk.Int, err error) {
	asset0Amount, ok := sdk.NewIntFromString(p.Assets[0].Amount)
	if !ok {
		return sdk.ZeroInt(), sdk.ZeroInt(), sdkerrors.Wrap(sdk.ErrIntOverflowCoin, "asset0")
	}

	asset1Amount, ok := sdk.NewIntFromString(p.Assets[1].Amount)
	if !ok {
		return sdk.ZeroInt(), sdk.ZeroInt(), sdkerrors.Wrap(sdk.ErrIntOverflowCoin, "asset1")
	}

	if p.Assets[0].Info.NativeToken != nil && p.Assets[1].Info.Token != nil {
		return asset0Amount, asset1Amount, nil
	} else if p.Assets[0].Info.Token != nil && p.Assets[1].Info.NativeToken != nil {
		return asset1Amount, asset0Amount, nil
	}

	return sdk.ZeroInt(), sdk.ZeroInt(), errors.New("can't be xatp")
}

type Cw20Balance struct {
	Balance string `json:"balance,omitempty"`
}
