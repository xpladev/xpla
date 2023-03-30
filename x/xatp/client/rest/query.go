package rest

import (
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"
	"github.com/xpladev/xpla/x/xatp/types"
)

func registerQueryRoutes(clientCtx client.Context, r *mux.Router) {
	// Get the current xatp parameter values
	r.HandleFunc(
		"/xatp/parameters",
		paramsHandlerFn(clientCtx),
	).Methods(http.MethodGet)

	r.HandleFunc(
		"/xatp/xatps",
		xatpsHandlerFn(clientCtx),
	).Methods(http.MethodGet)

	r.HandleFunc(
		"/xatp/xatp/{denom}",
		paramsHandlerFn(clientCtx),
	).Methods(http.MethodGet)

	// Get the amount held in the xatp pool
	r.HandleFunc(
		"/xatp/pool",
		xatpPoolHandler(clientCtx),
	).Methods(http.MethodGet)
}

// HTTP request handler to query the distribution params values
func paramsHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryParams)
		res, height, err := clientCtx.QueryWithData(route, nil)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, res)
	}
}

func xatpsHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryXatps)
		res, height, err := clientCtx.QueryWithData(route, nil)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, res)
	}
}

func xatpHandlerFn(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		denom := vars["denom"]

		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		route := fmt.Sprintf("custom/%s/%s", types.QuerierRoute, types.QueryXatp)
		bz, err := clientCtx.LegacyAmino.MarshalJSON(types.QueryXatpRequest{Denom: denom})
		if rest.CheckBadRequestError(w, err) {
			return
		}
		res, height, err := clientCtx.QueryWithData(route, bz)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, res)
	}
}

func xatpPoolHandler(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		res, height, err := clientCtx.QueryWithData(fmt.Sprintf("custom/%s/xatp_pool", types.QuerierRoute), nil)
		if rest.CheckInternalServerError(w, err) {
			return
		}

		var result sdk.DecCoins
		if rest.CheckInternalServerError(w, clientCtx.LegacyAmino.UnmarshalJSON(res, &result)) {
			return
		}

		clientCtx = clientCtx.WithHeight(height)
		rest.PostProcessResponse(w, clientCtx, result)
	}
}
