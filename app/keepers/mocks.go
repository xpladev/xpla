package keepers

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
)

type MockErc20Keeper struct{}

func (ek MockErc20Keeper) GetERC20PrecompileInstance(
	ctx sdk.Context,
	address common.Address,
) (contract vm.PrecompiledContract, found bool, err error) {
	return nil, false, nil
}
