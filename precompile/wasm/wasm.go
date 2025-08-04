package wasm

import (
	"embed"
	"errors"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"

	cmn "github.com/cosmos/evm/precompiles/common"

	"github.com/xpladev/xpla/precompile/util"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ vm.PrecompiledContract = PrecompiledWasm{}

var (
	Address = common.HexToAddress(hexAddress)
	ABI     = abi.ABI{}

	//go:embed IWasm.abi
	abiFS embed.FS
)

type PrecompiledWasm struct {
	cmn.Precompile
	ak  AccountKeeper
	wms WasmMsgServer
	wk  WasmKeeper
}

func init() {
	var err error
	ABI, err = util.LoadABI(abiFS, abiFile)
	if err != nil {
		panic(err)
	}
}

func NewPrecompiledWasm(ak AccountKeeper, wms WasmMsgServer, wk WasmKeeper) PrecompiledWasm {
	p := PrecompiledWasm{
		Precompile: cmn.Precompile{
			ABI:                  ABI,
			KvGasConfig:          storetypes.KVGasConfig(),
			TransientKVGasConfig: storetypes.TransientGasConfig(),
		},
		ak:  ak,
		wms: wms,
		wk:  wk,
	}
	p.SetAddress(common.HexToAddress(hexAddress))

	return p
}

func (p PrecompiledWasm) RequiredGas(input []byte) uint64 {
	// NOTE: This check avoid panicking when trying to decode the method ID
	if len(input) < 4 {
		return 0
	}

	methodID := input[:4]

	method, err := p.MethodById(methodID)
	if err != nil {
		// This should never happen since this method is going to fail during Run
		return 0
	}

	return p.Precompile.RequiredGas(input, p.IsTransaction(method))
}

func (p PrecompiledWasm) Run(evm *vm.EVM, contract *vm.Contract, readOnly bool) (bz []byte, err error) {
	if contract.Gas < wasmtypes.DefaultInstanceCost {
		return nil, errors.New("insufficient gas")
	}

	ctx, stateDB, method, initialGas, args, err := p.RunSetup(evm, contract, readOnly, p.IsTransaction)
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	// Start the balance change handler before executing the precompile.
	p.GetBalanceHandler().BeforeBalanceChange(ctx)

	// This handles any out of gas errors that may occur during the execution of a precompile tx or query.
	// It avoids panics and returns the out of gas error so the EVM can continue gracefully.
	defer cmn.HandleGasError(ctx, contract, initialGas, &err)()

	switch MethodWasm(method.Name) {
	case InstantiateContract:
		bz, err = p.instantiateContract(ctx, stateDB, contract.Caller(), method, args)
	case InstantiateContract2:
		bz, err = p.instantiateContract2(ctx, stateDB, contract.Caller(), method, args)
	case ExecuteContract:
		bz, err = p.executeContract(ctx, stateDB, contract.Caller(), method, args)
	case MigrateContract:
		bz, err = p.migrateContract(ctx, stateDB, contract.Caller(), method, args)
	case SmartContractState:
		bz, err = p.smartContractState(ctx, method, args)
	default:
		bz, err = nil, errors.New("method not found")
	}
	if err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	cost := ctx.GasMeter().GasConsumed() - initialGas

	if !contract.UseGas(cost, nil, tracing.GasChangeCallPrecompiledContract) {
		return cmn.ReturnRevertError(evm, vm.ErrOutOfGas)
	}

	// Process the native balance changes after the method execution.
	if err = p.GetBalanceHandler().AfterBalanceChange(ctx, stateDB); err != nil {
		return cmn.ReturnRevertError(evm, err)
	}

	return bz, nil
}

func (PrecompiledWasm) IsTransaction(method *abi.Method) bool {
	switch MethodWasm(method.Name) {
	case InstantiateContract,
		InstantiateContract2,
		ExecuteContract,
		MigrateContract:
		return true
	default:
		return false
	}
}

func (p PrecompiledWasm) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("xpla evm extension", "wasm")
}

func (p PrecompiledWasm) instantiateContract(ctx sdk.Context, stateDB vm.StateDB, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

	fromAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(fromAddress, sender); err != nil {
		return nil, err
	}

	admin, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	codeId, err := util.GetBigInt(args[2])
	if err != nil {
		return nil, err
	}

	label, err := util.GetString(args[3])
	if err != nil {
		return nil, err
	}

	msg, err := util.GetByteArray(args[4])
	if err != nil {
		return nil, err
	}

	coins, err := util.GetCoins(args[5])
	if err != nil {
		return nil, err
	}

	instantiateMsg := &wasmtypes.MsgInstantiateContract{
		Sender: fromAddress.String(),
		Admin:  admin.String(),
		CodeID: codeId.Uint64(),
		Label:  label,
		Msg:    msg,
		Funds:  coins,
	}

	res, err := p.wms.InstantiateContract(ctx, instantiateMsg)
	if err != nil {
		return nil, err
	}

	cosmosContractAddress, err := sdk.AccAddressFromBech32(res.Address)
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(cosmosContractAddress.Bytes())

	err = p.EmitInstantiateContractEvent(ctx, stateDB, sender, common.BytesToAddress(admin.Bytes()), contractAddress, codeId.BigInt(), label, msg, coins, res.Data)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(contractAddress, res.Data)
}

func (p PrecompiledWasm) instantiateContract2(ctx sdk.Context, stateDB vm.StateDB, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

	fromAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(fromAddress, sender); err != nil {
		return nil, err
	}

	admin, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	codeId, err := util.GetBigInt(args[2])
	if err != nil {
		return nil, err
	}

	label, err := util.GetString(args[3])
	if err != nil {
		return nil, err
	}

	msg, err := util.GetByteArray(args[4])
	if err != nil {
		return nil, err
	}

	coins, err := util.GetCoins(args[5])
	if err != nil {
		return nil, err
	}

	salt, err := util.GetByteArray(args[6])
	if err != nil {
		return nil, err
	}

	fixMsg, err := util.GetBool(args[7])
	if err != nil {
		return nil, err
	}

	instantiate2Msg := &wasmtypes.MsgInstantiateContract2{
		Sender: fromAddress.String(),
		Admin:  admin.String(),
		CodeID: codeId.Uint64(),
		Label:  label,
		Msg:    msg,
		Funds:  coins,
		Salt:   salt,
		FixMsg: fixMsg,
	}

	res, err := p.wms.InstantiateContract2(ctx, instantiate2Msg)
	if err != nil {
		return nil, err
	}

	cosmosContractAddress, err := sdk.AccAddressFromBech32(res.Address)
	if err != nil {
		return nil, err
	}

	contractAddress := common.BytesToAddress(cosmosContractAddress.Bytes())

	err = p.EmitInstantiateContractEvent(ctx, stateDB, sender, common.BytesToAddress(admin.Bytes()), contractAddress, codeId.BigInt(), label, msg, coins, res.Data)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(contractAddress, res.Data)
}

func (p PrecompiledWasm) executeContract(ctx sdk.Context, stateDB vm.StateDB, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
	fromAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(fromAddress, sender); err != nil {
		return nil, err
	}

	contractAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	contractAccount := p.ak.GetAccount(ctx, contractAddress)

	msg, err := util.GetByteArray(args[2])
	if err != nil {
		return nil, err
	}

	coins, err := util.GetCoins(args[3])
	if err != nil {
		return nil, err
	}

	executeMsg := &wasmtypes.MsgExecuteContract{
		Sender:   fromAddress.String(),
		Contract: contractAccount.GetAddress().String(),
		Msg:      msg,
		Funds:    coins,
	}

	res, err := p.wms.ExecuteContract(ctx, executeMsg)
	if err != nil {
		return nil, err
	}

	err = p.EmitExecuteContractEvent(ctx, stateDB, sender, common.BytesToAddress(contractAddress.Bytes()), msg, coins, res.Data)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.Data)
}

func (p PrecompiledWasm) migrateContract(ctx sdk.Context, stateDB vm.StateDB, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

	fromAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	if err = util.ValidateSigner(fromAddress, sender); err != nil {
		return nil, err
	}

	contractAddress, err := util.GetAccAddress(args[1])
	if err != nil {
		return nil, err
	}

	contractAccount := p.ak.GetAccount(ctx, contractAddress)

	codeId, err := util.GetBigInt(args[2])
	if err != nil {
		return nil, err
	}

	msg, err := util.GetByteArray(args[3])
	if err != nil {
		return nil, err
	}

	migrateMsg := &wasmtypes.MsgMigrateContract{
		Sender:   fromAddress.String(),
		Contract: contractAccount.GetAddress().String(),
		CodeID:   codeId.Uint64(),
		Msg:      msg,
	}

	res, err := p.wms.MigrateContract(ctx, migrateMsg)
	if err != nil {
		return nil, err
	}

	err = p.EmitMigrateContractEvent(ctx, stateDB, sender, common.BytesToAddress(contractAddress.Bytes()), codeId.BigInt(), msg, res.Data)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res.Data)
}

func (p PrecompiledWasm) smartContractState(ctx sdk.Context, method *abi.Method, args []interface{}) ([]byte, error) {
	contractAddress, err := util.GetAccAddress(args[0])
	if err != nil {
		return nil, err
	}

	contractAccount := p.ak.GetAccount(ctx, contractAddress)

	queryData, err := util.GetByteArray(args[1])
	if err != nil {
		return nil, err
	}

	res, err := p.wk.QuerySmart(ctx, contractAccount.GetAddress(), queryData)
	if err != nil {
		return nil, err
	}

	return method.Outputs.Pack(res)
}
