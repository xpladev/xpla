package rest

import (
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	clientrest "github.com/cosmos/cosmos-sdk/client/rest"
	"github.com/cosmos/cosmos-sdk/client/tx"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	govrest "github.com/cosmos/cosmos-sdk/x/gov/client/rest"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/gorilla/mux"
	"github.com/xpladev/xpla/x/specialvalidator/types"
)

func RegisterHandlers(cliecntCtx client.Context, rtr *mux.Router) {
	r := clientrest.WithHTTPDeprecationHeaders(rtr)

	registerQueryRoutes(cliecntCtx, r)
}

func RegisterSpecialValidatorProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "register_special_validator",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req RegisterSpecialValidatorProposalReq
			if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
				return
			}

			req.BaseReq = req.BaseReq.Sanitize()
			if !req.BaseReq.ValidateBasic(w) {
				return
			}

			from := sdk.MustAccAddressFromBech32(req.BaseReq.From)

			var pubKey cryptotypes.PubKey
			if err := clientCtx.Codec.UnmarshalInterfaceJSON([]byte(req.Pubkey), &pubKey); err != nil {
				rest.CheckBadRequestError(w, err)
				return
			}

			content, err := types.NewRegisterSpecialValidatorProposal(req.Title, req.Description, from, sdk.ValAddress(from), pubKey, req.Amount, req.ValidatorDescription)
			if rest.CheckBadRequestError(w, err) {
				return
			}

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

func UnregisterSpecialValidatorProposalRESTHandler(clientCtx client.Context) govrest.ProposalRESTHandler {
	return govrest.ProposalRESTHandler{
		SubRoute: "unregister_special_validator",
		Handler: func(w http.ResponseWriter, r *http.Request) {
			var req UnregisterSpecialValidatorProposalReq
			if !rest.ReadRESTReq(w, r, clientCtx.LegacyAmino, &req) {
				return
			}

			req.BaseReq = req.BaseReq.Sanitize()
			if !req.BaseReq.ValidateBasic(w) {
				return
			}

			content := types.NewUnregisterSpecialValidatorProposal(req.Title, req.Description, req.ValidatorAddress)

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
