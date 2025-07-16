package types

import (
	sdkerrors "cosmossdk.io/errors"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

var (
	ErrErc20Transfer    = sdkerrors.Register(banktypes.ModuleName, 1001, "fail to transfer erc20")
	ErrErc20Balance     = sdkerrors.Register(banktypes.ModuleName, 1002, "fail to query balance erc20")
	ErrErc20TotalSupply = sdkerrors.Register(banktypes.ModuleName, 1003, "fail to query total supply erc20")
)
