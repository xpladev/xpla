package precompile

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/cosmos/evm/precompiles/common"
	erc20types "github.com/cosmos/evm/x/erc20/types"
	"github.com/cosmos/evm/x/vm/statedb"
)

// MockERC20Keeper is a no-op ERC20Keeper for chains that do not use the ERC20 module.
// It implements common.ERC20Keeper so that ics20 precompile can be constructed;
// ICS20 transfer will always use regular transfer (no ERC20 conversion).
type MockERC20Keeper struct{}

var _ common.ERC20Keeper = (*MockERC20Keeper)(nil)

func (MockERC20Keeper) GetCoinAddress(ctx sdk.Context, denom string) (ethcommon.Address, error) {
	return ethcommon.Address{}, nil
}

func (MockERC20Keeper) GetERC20Map(ctx sdk.Context, erc20 ethcommon.Address) []byte {
	return nil
}

func (MockERC20Keeper) GetTokenPair(ctx sdk.Context, id []byte) (erc20types.TokenPair, bool) {
	return erc20types.TokenPair{}, false
}

func (MockERC20Keeper) IsERC20Enabled(ctx sdk.Context) bool {
	return false
}

func (MockERC20Keeper) GetTokenPairID(ctx sdk.Context, token string) []byte {
	return nil
}

func (MockERC20Keeper) ConvertERC20IntoCoinsForNativeToken(ctx sdk.Context, stateDB *statedb.StateDB, contract ethcommon.Address, amount math.Int, receiver sdk.AccAddress, sender ethcommon.Address, commit bool, callFromPrecompile bool) (*erc20types.MsgConvertERC20Response, error) {
	return nil, nil
}
