package types

import (
	"cosmossdk.io/math"

	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/evm/evmd/eips"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func SetConfig() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount(Bech32PrefixAccAddr, Bech32PrefixAccPub)
	config.SetBech32PrefixForValidator(Bech32PrefixValAddr, Bech32PrefixValPub)
	config.SetBech32PrefixForConsensusNode(Bech32PrefixConsAddr, Bech32PrefixConsPub)
	config.SetCoinType(CoinType)
	config.SetFullFundraiserPath(FullFundraiserPath)
	config.Seal()
}

const (
	// base chain coin info for xpla network
	XplaBaseDenom    = "axpla"
	XplaDisplayDenom = "xpla"
	XplaDecimals     = evmtypes.EighteenDecimals
)

// cosmosEVMActivators defines a map of opcode modifiers associated
// with a key defining the corresponding EIP.
var cosmosEVMActivators = map[int]func(*vm.JumpTable){
	0o000: eips.Enable0000,
	0o001: eips.Enable0001,
	0o002: eips.Enable0002,
}

// EVMOptionsFn defines a function type for setting app options specifically for
// the Cosmos EVM app. The function should receive the chainID and return an error if
// any.
type EVMOptionsFn func(uint64) error

// NoOpEVMOptions is a no-op function that can be used when the app does not
// need any specific configuration.
func NoOpEVMOptions(_ uint64) error {
	return nil
}

var sealed = false

// EvmAppOptions allows to setup the global configuration
// for the Cosmos EVM chain.
func EvmAppOptions(chainID uint64) error {
	if sealed {
		return nil
	}

	coinInfo := evmtypes.EvmCoinInfo{
		Denom:         XplaBaseDenom,
		ExtendedDenom: XplaBaseDenom,
		DisplayDenom:  XplaDisplayDenom,
		Decimals:      XplaDecimals,
	}

	// set the denom info for the chain
	if err := setBaseDenom(coinInfo); err != nil {
		return err
	}

	ethCfg := evmtypes.DefaultChainConfig(chainID)

	err := evmtypes.NewEVMConfigurator().
		WithExtendedEips(cosmosEVMActivators).
		WithChainConfig(ethCfg).
		WithEVMCoinInfo(coinInfo).
		Configure()
	if err != nil {
		return err
	}

	sealed = true
	return nil
}

// setBaseDenom registers the display denom and base denom and sets the
// base denom for the chain.
func setBaseDenom(ci evmtypes.EvmCoinInfo) error {
	if err := sdk.RegisterDenom(ci.DisplayDenom, math.LegacyOneDec()); err != nil {
		return err
	}

	// sdk.RegisterDenom will automatically overwrite the base denom when the
	// new setBaseDenom() are lower than the current base denom's units.
	return sdk.RegisterDenom(ci.Denom, math.LegacyNewDecWithPrec(1, int64(ci.Decimals)))
}
