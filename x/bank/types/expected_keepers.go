package types

import (
	"context"
	"math/big"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/vm"
	evmtypes "github.com/xpladev/ethermint/x/evm/types"
)

// TestErcKeeper defines the expected interface for the ERC20 module.
type TestErcKeeper interface {
	QueryBalanceOf(ctx sdk.Context, contractAddress common.Address, account sdk.AccAddress) (sdkmath.Int, error)
	QueryTotalSupply(ctx sdk.Context, contractAddress common.Address) (sdkmath.Int, error)
	ExecuteTransfer(ctx sdk.Context, contractAddress common.Address, sender, to sdk.AccAddress, amount *big.Int) error
}

// TestErcKeeper defines the expected interface for the ERC20 module.
type EvmKeeper interface {
	ApplyMessage(ctx sdk.Context, msg core.Message, tracer vm.EVMLogger, commit bool) (*evmtypes.MsgEthereumTxResponse, error)
	EstimateGas(c context.Context, req *evmtypes.EthCallRequest) (*evmtypes.EstimateGasResponse, error)
	GetNonce(ctx sdk.Context, addr common.Address) uint64
}
