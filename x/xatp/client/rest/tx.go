package rest

import (
	"net/http"

	"github.com/gorilla/mux"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"

	"github.com/xpladev/xpla/x/xatp/types"
)

type (
	fundXatpPoolReq struct {
		BaseReq rest.BaseReq `json:"base_req"`
		Amount  sdk.Coins    `json:"amount"`
	}
)

func registerTxHandlers(clientCtx client.Context, r *mux.Router) {
	// Fund the community pool
	r.HandleFunc(
		"/xatp/xatp_pool",
		newFundXatpPoolHandlerFn(clientCtx),
	).Methods("POST")
}

func newFundXatpPoolHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req fundXatpPoolReq
		if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
			return
		}

		req.BaseReq = req.BaseReq.Sanitize()
		if !req.BaseReq.ValidateBasic(w) {
			return
		}

		fromAddr, err := sdk.AccAddressFromBech32(req.BaseReq.From)
		if rest.CheckBadRequestError(w, err) {
			return
		}

		msg := types.NewMsgFundXatpPool(req.Amount, fromAddr)
		if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
			return
		}

		tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
	}
}
