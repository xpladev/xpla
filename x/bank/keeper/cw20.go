package keeper

import (
	"context"
	"encoding/json"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/xpladev/xpla/x/bank/types"
)

type Cw20Keeper struct {
	wk  types.WasmKeeper
	wmk types.WasmMsgServer
}

func NewCw20Keeper(wk types.WasmKeeper, wmk types.WasmMsgServer) Cw20Keeper {
	return Cw20Keeper{
		wk:  wk,
		wmk: wmk,
	}
}

func (ck Cw20Keeper) QueryTokenInfo(goCtx context.Context, contractAddress sdk.AccAddress) (*types.TokenInfoResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	rawQueryData, err := json.Marshal(map[string]any{"token_info": types.QueryMsg_TokenInfo{}})
	if err != nil {
		return nil, err
	}

	rawResponseData, err := ck.wk.QuerySmart(ctx, contractAddress, rawQueryData)
	if err != nil {
		return nil, err
	}

	var response types.TokenInfoResponse
	if err := json.Unmarshal(rawResponseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (ck Cw20Keeper) QueryBalance(goCtx context.Context, contractAddress sdk.AccAddress, req *types.QueryMsg_Balance) (*types.BalanceResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	rawQueryData, err := json.Marshal(map[string]any{"balance": req})
	if err != nil {
		return nil, err
	}

	rawResponseData, err := ck.wk.QuerySmart(ctx, contractAddress, rawQueryData)
	if err != nil {
		return nil, err
	}

	var response types.BalanceResponse
	if err := json.Unmarshal(rawResponseData, &response); err != nil {
		return nil, err
	}

	return &response, nil
}

func (ck Cw20Keeper) ExecuteTransfer(goCtx context.Context, sender sdk.AccAddress, contractAddress sdk.AccAddress, req *types.ExecuteMsg_Transfer) (*wasmtypes.MsgExecuteContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	rawExecuteData, err := json.Marshal(map[string]any{"transfer": req})
	if err != nil {
		return nil, err
	}

	msg := &wasmtypes.MsgExecuteContract{
		Sender:   sender.String(),
		Contract: contractAddress.String(),
		Msg:      rawExecuteData,
		Funds:    sdk.NewCoins(),
	}

	return ck.wmk.ExecuteContract(ctx, msg)
}
