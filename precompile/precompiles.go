package precompile

import (
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"cosmossdk.io/core/address"

	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	"github.com/cosmos/evm/precompiles/bech32"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"

	ibctransferkeeper "github.com/cosmos/ibc-go/v10/modules/apps/transfer/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v10/modules/core/04-channel/keeper"

	pauth "github.com/xpladev/xpla/precompile/auth"
	pbank "github.com/xpladev/xpla/precompile/bank"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
	xplabankkeeper "github.com/xpladev/xpla/x/bank/keeper"
)

const bech32PrecompileBaseGas = 6_000

type Optionals struct {
	AddressCodec       address.Codec // used by gov/staking
	ValidatorAddrCodec address.Codec // used by slashing
	ConsensusAddrCodec address.Codec // used by slashing
}

var PrecompiledAddressesXpla = []common.Address{
	pbank.Address, pwasm.Address, pauth.Address,
}

// NewAvailableStaticPrecompiles returns the list of all available static precompiled contracts from Cosmos EVM.
// NOTE: this should only be used during initialization of the Keeper.
func NewAvailableStaticPrecompiles(
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	transferKeeper ibctransferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
	evmKeeper *evmkeeper.Keeper,
	govKeeper govkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	ak pwasm.AccountKeeper,
	bk xplabankkeeper.Keeper,
	wms pwasm.WasmMsgServer,
	wk pwasm.WasmKeeper,
	authAk pauth.AccountKeeper,
	codec codec.Codec,
	opts ...Option,
) map[common.Address]vm.PrecompiledContract {
	options := defaultOptionals()
	for _, opt := range opts {
		opt(&options)
	}

	// Clone the mapping from the latest EVM fork.
	precompiles := maps.Clone(vm.PrecompiledContractsPrague)

	// secp256r1 precompile as per EIP-7212
	p256Precompile := &p256.Precompile{}

	bech32Precompile, err := bech32.NewPrecompile(bech32PrecompileBaseGas)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate bech32 precompile: %w", err))
	}

	stakingPrecompile, err := stakingprecompile.NewPrecompile(
		stakingKeeper,
		stakingkeeper.NewMsgServerImpl(&stakingKeeper),
		stakingkeeper.NewQuerier(&stakingKeeper),
		bk,
		options.AddressCodec,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate staking precompile: %w", err))
	}

	distributionPrecompile, err := distprecompile.NewPrecompile(
		distributionKeeper,
		distributionkeeper.NewMsgServerImpl(distributionKeeper),
		distributionkeeper.NewQuerier(distributionKeeper),
		stakingKeeper,
		bk,
		options.AddressCodec,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate distribution precompile: %w", err))
	}

	ibcTransferPrecompile, err := ics20precompile.NewPrecompile(
		bk,
		stakingKeeper,
		transferKeeper,
		channelKeeper,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate ICS20 precompile: %w", err))
	}

	govPrecompile, err := govprecompile.NewPrecompile(
		govkeeper.NewMsgServerImpl(&govKeeper),
		govkeeper.NewQueryServer(&govKeeper),
		bk,
		codec,
		options.AddressCodec,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate gov precompile: %w", err))
	}

	slashingPrecompile, err := slashingprecompile.NewPrecompile(
		slashingKeeper,
		slashingkeeper.NewMsgServerImpl(slashingKeeper),
		bk,
		options.ValidatorAddrCodec,
		options.ConsensusAddrCodec,
	)
	if err != nil {
		panic(fmt.Errorf("failed to instantiate slashing precompile: %w", err))
	}

	// Stateless precompiles
	precompiles[bech32Precompile.Address()] = bech32Precompile
	precompiles[p256Precompile.Address()] = p256Precompile

	// Stateful precompiles
	precompiles[stakingPrecompile.Address()] = stakingPrecompile
	precompiles[distributionPrecompile.Address()] = distributionPrecompile
	precompiles[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	precompiles[govPrecompile.Address()] = govPrecompile
	precompiles[slashingPrecompile.Address()] = slashingPrecompile

	// xpla precompiles
	precompiles[pbank.Address] = pbank.NewPrecompiledBank(bk)
	precompiles[pwasm.Address] = pwasm.NewPrecompiledWasm(ak, wms, wk, bk)
	precompiles[pauth.Address] = pauth.NewPrecompiledAuth(authAk)

	return precompiles
}

func defaultOptionals() Optionals {
	return Optionals{
		AddressCodec:       addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddrCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddrCodec: addresscodec.NewBech32Codec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}

type Option func(*Optionals)

func WithAddressCodec(c address.Codec) Option {
	return func(o *Optionals) { o.AddressCodec = c }
}

func WithValidatorAddrCodec(c address.Codec) Option {
	return func(o *Optionals) { o.ValidatorAddrCodec = c }
}

func WithConsensusAddrCodec(c address.Codec) Option {
	return func(o *Optionals) { o.ConsensusAddrCodec = c }
}
