package rest

import (
	"fmt"
	"net/http"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/types/rest"
	"github.com/gorilla/mux"
	"github.com/xpladev/xpla/x/reward/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func registerQueryRoutes(clientCtx client.Context, r *mux.Router) {
	// Get the current reward parameter values
	r.HandleFunc(
		"/reward/parameters",
		paramsHandlerFn(clientCtx),
	).Methods(http.MethodGet)

	// Get the amount held in the reward pool
	r.HandleFunc(
		"/reward/pool",
		rewardPoolHandler(clientCtx),
	).Methods(http.MethodGet)
}

// HTTP request handler to query the reward params values
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

func rewardPoolHandler(clientCtx client.Context) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		clientCtx, ok := rest.ParseQueryHeightOrReturnBadRequest(w, clientCtx, r)
		if !ok {
			return
		}

		res, height, err := clientCtx.QueryWithData(fmt.Sprintf("custom/%s/pool", types.QuerierRoute), nil)
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
