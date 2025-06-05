package types

import (
	"strings"

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
	// base chain ids for xpla network
	XplaBaseDenom    = "axpla"
	XplaDisplayDenom = "xpla"

	// base chain ids for xpla network
	XplaMainnetChainID   = "dimension_37"
	XplaTestnetChainID   = "cube_47"
	XplaHypercubeChainID = "hypercube_270"

	// default value
	DefaultBaseDenom    = "axpla"
	DefaultDisplayDenom = "xpla"
	DefaultChainID      = "test_9999"
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
type EVMOptionsFn func(string) error

// NoOpEVMOptions is a no-op function that can be used when the app does not
// need any specific configuration.
func NoOpEVMOptions(_ string) error {
	return nil
}

var sealed = false

// ChainsCoinInfo is a map of the chain id and its corresponding EvmCoinInfo
// that allows initializing the app with different coin info based on the
// chain id
var ChainsCoinInfo = map[string]evmtypes.EvmCoinInfo{
	XplaMainnetChainID: {
		Denom:         XplaBaseDenom,
		ExtendedDenom: XplaBaseDenom,
		DisplayDenom:  XplaDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals,
	},
	XplaTestnetChainID: {
		Denom:         XplaBaseDenom,
		ExtendedDenom: XplaBaseDenom,
		DisplayDenom:  XplaDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals,
	},
	XplaHypercubeChainID: {
		Denom:         XplaBaseDenom,
		ExtendedDenom: XplaBaseDenom,
		DisplayDenom:  XplaDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals,
	},
	DefaultChainID: {
		Denom:         DefaultBaseDenom,
		ExtendedDenom: DefaultBaseDenom,
		DisplayDenom:  DefaultDisplayDenom,
		Decimals:      evmtypes.EighteenDecimals,
	},
}

// EvmAppOptions allows to setup the global configuration
// for the Cosmos EVM chain.
func EvmAppOptions(chainID string) error {
	if sealed {
		return nil
	}

	id := strings.Split(chainID, "-")[0]
	coinInfo, found := ChainsCoinInfo[id]
	if !found {
		coinInfo, _ = ChainsCoinInfo[DefaultChainID]
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
