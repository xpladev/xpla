package keeper

import (
	"context"

	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/xpladev/xpla/x/auth/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ authtypes.QueryServer = queryServer{}

type queryServer struct {
	authtypes.QueryServer
	accountKeeper AccountKeeper
	wasmKeeper    types.WasmKeeper
}

// NewQueryServer extends only the SDK account queries with Wasm alias compatibility.
func NewQueryServer(accountKeeper AccountKeeper, wasmKeeper types.WasmKeeper) authtypes.QueryServer {
	return queryServer{
		QueryServer:   authkeeper.NewQueryServer(accountKeeper.AccountKeeper),
		accountKeeper: accountKeeper,
		wasmKeeper:    wasmKeeper,
	}
}

// canonicalAddress leaves invalid and absent addresses to the SDK query server,
// preserving its validation and not-found errors.
func (s queryServer) canonicalAddress(ctx context.Context, requested string) (string, error) {
	addr, err := s.accountKeeper.AddressCodec().StringToBytes(requested)
	if err != nil || requested == "" {
		return requested, nil
	}
	resolved, err := s.accountKeeper.ResolveAccountAddress(ctx, addr, s.wasmKeeper)
	if err != nil {
		return "", status.Errorf(codes.Internal, "resolve account: %v", err)
	}
	if resolved == nil {
		return requested, nil
	}
	canonical, err := s.accountKeeper.AddressCodec().BytesToString(resolved)
	if err != nil {
		return "", status.Errorf(codes.Internal, "resolve account: %v", err)
	}
	return canonical, nil
}

func (s queryServer) Account(ctx context.Context, req *authtypes.QueryAccountRequest) (*authtypes.QueryAccountResponse, error) {
	if req == nil {
		return s.QueryServer.Account(ctx, req)
	}
	canonical, err := s.canonicalAddress(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	resolved := *req
	resolved.Address = canonical
	return s.QueryServer.Account(ctx, &resolved)
}

func (s queryServer) AccountInfo(ctx context.Context, req *authtypes.QueryAccountInfoRequest) (*authtypes.QueryAccountInfoResponse, error) {
	if req == nil {
		return s.QueryServer.AccountInfo(ctx, req)
	}
	canonical, err := s.canonicalAddress(ctx, req.Address)
	if err != nil {
		return nil, err
	}
	resolved := *req
	resolved.Address = canonical
	response, err := s.QueryServer.AccountInfo(ctx, &resolved)
	if err == nil {
		response.Info.Address = req.Address
	}
	return response, err
}
