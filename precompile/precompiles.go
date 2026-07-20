package precompile

import (
	"fmt"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distributionkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	govkeeper "github.com/cosmos/cosmos-sdk/x/gov/keeper"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	evmaddress "github.com/cosmos/evm/encoding/address"
	ibcutils "github.com/cosmos/evm/ibc"
	"github.com/cosmos/evm/precompiles/bech32"
	distprecompile "github.com/cosmos/evm/precompiles/distribution"
	govprecompile "github.com/cosmos/evm/precompiles/gov"
	ics02precompile "github.com/cosmos/evm/precompiles/ics02"
	ics20precompile "github.com/cosmos/evm/precompiles/ics20"
	"github.com/cosmos/evm/precompiles/p256"
	slashingprecompile "github.com/cosmos/evm/precompiles/slashing"
	stakingprecompile "github.com/cosmos/evm/precompiles/staking"
	evmprecompiletypes "github.com/cosmos/evm/precompiles/types"
	evmkeeper "github.com/cosmos/evm/x/vm/keeper"
	"github.com/cosmos/evm/x/vm/statedb"
	evmtypes "github.com/cosmos/evm/x/vm/types"

	ibctransferkeeper "github.com/cosmos/ibc-go/v11/modules/apps/transfer/keeper"
	channelkeeper "github.com/cosmos/ibc-go/v11/modules/core/04-channel/keeper"

	pauth "github.com/xpladev/xpla/precompile/auth"
	pbank "github.com/xpladev/xpla/precompile/bank"
	pwasm "github.com/xpladev/xpla/precompile/wasm"
	xplabankkeeper "github.com/xpladev/xpla/x/bank/keeper"
	xbanktypes "github.com/xpladev/xpla/x/bank/types"
)

const bech32PrecompileBaseGas = 6_000

var PrecompiledAddressesXpla = []common.Address{
	pbank.Address,
	pwasm.Address,
	pauth.Address,
	pwasm.DelegatecallAddress,
}

func registeredStaticPrecompileAddresses() []common.Address {
	return []common.Address{
		common.HexToAddress(evmtypes.P256PrecompileAddress),
		common.HexToAddress(evmtypes.Bech32PrecompileAddress),
		common.HexToAddress(evmtypes.StakingPrecompileAddress),
		common.HexToAddress(evmtypes.DistributionPrecompileAddress),
		common.HexToAddress(evmtypes.ICS20PrecompileAddress),
		common.HexToAddress(evmtypes.GovPrecompileAddress),
		common.HexToAddress(evmtypes.SlashingPrecompileAddress),
		common.HexToAddress(evmtypes.ICS02PrecompileAddress),
		pbank.Address,
		pwasm.Address,
		pauth.Address,
		pwasm.DelegatecallAddress,
	}
}

// DefaultActiveStaticPrecompiles returns XPLA's default active static precompile set.
func DefaultActiveStaticPrecompiles() []string {
	activePrecompiles := []string{
		evmtypes.P256PrecompileAddress,
		evmtypes.Bech32PrecompileAddress,
		evmtypes.StakingPrecompileAddress,
		evmtypes.DistributionPrecompileAddress,
		evmtypes.ICS20PrecompileAddress,
		evmtypes.GovPrecompileAddress,
		evmtypes.SlashingPrecompileAddress,
		evmtypes.ICS02PrecompileAddress,
	}

	for _, precompile := range PrecompiledAddressesXpla {
		activePrecompiles = append(activePrecompiles, precompile.Hex())
	}

	return activePrecompiles
}

// ValidateActiveStaticPrecompiles rejects active precompiles that XPLA does not
// register in the EVM keeper. Upstream validation only checks address format,
// duplicates, and sorting; allowing an unregistered address can panic at runtime.
func ValidateActiveStaticPrecompiles(precompiles []string) error {
	if err := evmtypes.ValidatePrecompiles(precompiles); err != nil {
		return err
	}

	supported := make(map[common.Address]struct{}, len(vm.PrecompiledContractsPrague)+len(registeredStaticPrecompileAddresses()))
	for address := range vm.PrecompiledContractsPrague {
		supported[address] = struct{}{}
	}
	for _, address := range registeredStaticPrecompileAddresses() {
		supported[address] = struct{}{}
	}

	for _, precompile := range precompiles {
		if _, ok := supported[common.HexToAddress(precompile)]; !ok {
			return fmt.Errorf("unsupported active static precompile: %s", precompile)
		}
	}

	return nil
}

type wasmDelegatePrecompile struct {
	*pwasm.PrecompiledWasm
}

func (p wasmDelegatePrecompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	return p.PrecompiledWasm.RunDelegate(evm, contract, readOnly)
}

type ics20Precompile struct {
	*ics20precompile.Precompile
}

func (p ics20Precompile) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) ([]byte, error) {
	return p.RunNativeAction(evm, contract, func(ctx sdk.Context) ([]byte, error) {
		ctx = xbanktypes.WithEVMStateDB(ctx, evm.StateDB.(*statedb.StateDB))
		return p.Execute(ctx, evm.StateDB, contract, readOnly)
	})
}

// NewAvailableStaticPrecompiles returns the list of all available static precompiled contracts from Cosmos EVM.
// NOTE: this should only be used during initialization of the Keeper.
func NewAvailableStaticPrecompiles(
	stakingKeeper stakingkeeper.Keeper,
	distributionKeeper distributionkeeper.Keeper,
	transferKeeper *ibctransferkeeper.Keeper,
	channelKeeper *channelkeeper.Keeper,
	clientKeeper ibcutils.ClientKeeper,
	evmKeeper *evmkeeper.Keeper,
	govKeeper govkeeper.Keeper,
	slashingKeeper slashingkeeper.Keeper,
	ak pwasm.AccountKeeper,
	bk xplabankkeeper.Keeper,
	wms pwasm.WasmMsgServer,
	wk pwasm.WasmKeeper,
	authAk pauth.AccountKeeper,
	codec codec.Codec,
	opts ...evmprecompiletypes.Option,
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

	stakingPrecompile := stakingprecompile.NewPrecompile(
		stakingKeeper,
		stakingkeeper.NewMsgServerImpl(&stakingKeeper),
		stakingkeeper.NewQuerier(&stakingKeeper),
		bk,
		options.AddressCodec,
	)

	distributionPrecompile := distprecompile.NewPrecompile(
		distributionKeeper,
		distributionkeeper.NewMsgServerImpl(distributionKeeper),
		distributionkeeper.NewQuerier(distributionKeeper),
		stakingKeeper,
		bk,
		options.AddressCodec,
	)

	ibcTransferPrecompile := ics20Precompile{Precompile: ics20precompile.NewPrecompile(
		bk,
		stakingKeeper,
		transferKeeper,
		channelKeeper,
		MockERC20Keeper{},
	)}
	ibcClientPrecompile := ics02precompile.NewPrecompile(
		codec,
		clientKeeper,
	)

	govPrecompile := govprecompile.NewPrecompile(
		govkeeper.NewMsgServerImpl(&govKeeper),
		govkeeper.NewQueryServer(&govKeeper),
		bk,
		codec,
		options.AddressCodec,
	)

	slashingPrecompile := slashingprecompile.NewPrecompile(
		slashingKeeper,
		slashingkeeper.NewMsgServerImpl(slashingKeeper),
		bk,
		options.ValidatorAddrCodec,
		options.ConsensusAddrCodec,
	)

	// Stateless precompiles
	precompiles[bech32Precompile.Address()] = bech32Precompile
	precompiles[p256Precompile.Address()] = p256Precompile

	// Stateful precompiles
	precompiles[stakingPrecompile.Address()] = stakingPrecompile
	precompiles[distributionPrecompile.Address()] = distributionPrecompile
	precompiles[ibcClientPrecompile.Address()] = ibcClientPrecompile
	precompiles[ibcTransferPrecompile.Address()] = ibcTransferPrecompile
	precompiles[govPrecompile.Address()] = govPrecompile
	precompiles[slashingPrecompile.Address()] = slashingPrecompile

	// xpla precompiles
	precompiles[pbank.Address] = pbank.NewPrecompiledBank(bk)
	precompileWasm := pwasm.NewPrecompiledWasm(ak, wms, wk, bk)
	precompiles[pwasm.Address] = precompileWasm
	precompiles[pauth.Address] = pauth.NewPrecompiledAuth(authAk)
	// delegatecall wasm
	precompiles[pwasm.DelegatecallAddress] = wasmDelegatePrecompile{PrecompiledWasm: precompileWasm}

	return precompiles
}

func defaultOptionals() evmprecompiletypes.Optionals {
	return evmprecompiletypes.Optionals{
		AddressCodec:       evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32AccountAddrPrefix()),
		ValidatorAddrCodec: evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ValidatorAddrPrefix()),
		ConsensusAddrCodec: evmaddress.NewEvmCodec(sdk.GetConfig().GetBech32ConsensusAddrPrefix()),
	}
}
