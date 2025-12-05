package keeper

import (
	"embed"
	"math/big"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	precompileutil "github.com/xpladev/xpla/precompile/util"
	"github.com/xpladev/xpla/x/bank/types"
)

const abiFile = "IERC20.abi"

var (
	ABI = abi.ABI{}

	//go:embed IERC20.abi
	abiFS embed.FS
)

type Erc20Keeper struct {
	ak banktypes.AccountKeeper
	ek types.EvmKeeper
}

func init() {
	var err error
	ABI, err = precompileutil.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewErc20Keeper(ak banktypes.AccountKeeper, ek types.EvmKeeper) Erc20Keeper {
	return Erc20Keeper{
		ak: ak,
		ek: ek,
	}
}

func (k Erc20Keeper) QueryTotalSupply(ctx sdk.Context, contractAddress common.Address) (sdkmath.Int, error) {
	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())

	res, err := k.ek.CallEVM(ctx, ABI, moduleAddress, contractAddress, false, nil, types.GetErc20Method(types.TotalSupply))
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.TotalSupply), res.Return())
	if err != nil || len(unpacked) == 0 {
		return sdkmath.ZeroInt(), err
	}

	bigTotalSupply, ok := unpacked[0].(*big.Int)
	if !ok {
		return sdkmath.ZeroInt(), types.ErrErc20TotalSupply
	}

	totalSupply := sdkmath.NewIntFromBigInt(bigTotalSupply)

	return totalSupply, nil
}

func (k Erc20Keeper) QueryBalanceOf(ctx sdk.Context, contractAddress common.Address, account sdk.AccAddress) (sdkmath.Int, error) {
	moduleAccount := k.ak.GetModuleAccount(ctx, banktypes.ModuleName)
	moduleAddress := common.BytesToAddress(moduleAccount.GetAddress().Bytes())
	ethAccount := common.BytesToAddress(account.Bytes())

	res, err := k.ek.CallEVM(ctx, ABI, moduleAddress, contractAddress, false, nil, types.GetErc20Method(types.BalanceOf), ethAccount)
	if err != nil {
		return sdkmath.ZeroInt(), err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.BalanceOf), res.Return())
	if err != nil || len(unpacked) == 0 {
		return sdkmath.ZeroInt(), err
	}

	bigBalance, ok := unpacked[0].(*big.Int)
	if !ok {
		return sdkmath.ZeroInt(), types.ErrErc20Balance
	}

	balance := sdkmath.NewIntFromBigInt(bigBalance)

	return balance, nil
}

func (k Erc20Keeper) ExecuteTransfer(ctx sdk.Context, contractAddress common.Address, sender, to sdk.AccAddress, amount *big.Int) error {
	ethSender := common.BytesToAddress(sender.Bytes())
	ethTo := common.BytesToAddress(to.Bytes())

	res, err := k.ek.CallEVM(ctx, ABI, ethSender, contractAddress, true, nil, types.GetErc20Method(types.Transfer), ethTo, amount)
	if err != nil {
		return err
	}

	unpacked, err := ABI.Unpack(types.GetErc20Method(types.Transfer), res.Return())
	if err != nil {
		return err
	}

	if len(unpacked) == 0 || !unpacked[0].(bool) {
		return types.ErrErc20Transfer
	}

	return nil
}
