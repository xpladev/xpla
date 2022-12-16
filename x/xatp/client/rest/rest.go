package rest

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/xpladev/xpla/x/xatp/types"

	"github.com/cosmos/cosmos-sdk/client"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func RegisterHandlers(clientCtx client.Context, rtr *mux.Router) {
	r := clientrest.WithHTTPDeprecationHeaders(rtr)

	registerQueryRoutes(clientCtx, r)
}

func RegisterXatpProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "register_xatp",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req RegisterXatpProposalReq
			if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
				return
			}

			req.BaseReq = req.BaseReq.Sanitize()
			if !req.BaseReq.ValidateBasic(w) {
				return
			}

			content := types.NewRegisterXatpProposal(req.Title, req.Description, req.Xatp.Token, req.Xatp.Pair, req.Xatp.Denom, int(req.Xatp.Decimals))

			msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
			if rest.CheckBadRequestError(w, err) {
				return
			}
			if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
				return
			}

			tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
		},
	}
}

func UnregisterXatpProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unregister_xatp",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req UnregisterXatpProposalReq
			if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
				return
			}

			req.BaseReq = req.BaseReq.Sanitize()
			if !req.BaseReq.ValidateBasic(w) {
				return
			}

			content := types.NewUnregisterXatpProposal(req.Title, req.Description, req.Denom)

			msg, err := govtypes.NewMsgSubmitProposal(content, req.Deposit, req.Proposer)
			if rest.CheckBadRequestError(w, err) {
				return
			}
			if rest.CheckBadRequestError(w, msg.ValidateBasic()) {
				return
			}

			tx.WriteGeneratedTxResponse(clientCtx, w, req.BaseReq, msg)
		},
	}
}
