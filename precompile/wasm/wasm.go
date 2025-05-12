package wasm

import (
	"embed"
	"errors"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/xpladev/ethermint/x/evm/statedb"

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
	return PrecompiledWasm{
		ak:  ak,
		wms: wms,
		wk:  wk,
	}
}

func (p PrecompiledWasm) RequiredGas(input []byte) uint64 {
	// Implement the method as needed
	return 0
}

func (p PrecompiledWasm) Run(evm *vm.EVM, input []byte) ([]byte, error) {
	method, argsBz := util.SplitInput(input)

	abiMethod, err := ABI.MethodById(method)
	if err != nil {
		return nil, err
	}

	args, err := abiMethod.Inputs.Unpack(argsBz)
	if err != nil {
		return nil, err
	}

	ctx := evm.StateDB.(*statedb.StateDB).GetContext()

	switch MethodWasm(abiMethod.Name) {
	case InstantiateContract:
		return p.instantiateContract(ctx, evm.Origin, abiMethod, args)
	case InstantiateContract2:
		return p.instantiateContract2(ctx, evm.Origin, abiMethod, args)
	case ExecuteContract:
		return p.executeContract(ctx, evm.Origin, abiMethod, args)
	case MigrateContract:
		return p.migrateContract(ctx, evm.Origin, abiMethod, args)
	case SmartContractState:
		return p.smartContractState(ctx, abiMethod, args)
	default:
		return nil, errors.New("method not found")
	}
}

func (p PrecompiledWasm) instantiateContract(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

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

	return method.Outputs.Pack(contractAddress, res.Data)
}

func (p PrecompiledWasm) instantiateContract2(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

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

	return method.Outputs.Pack(contractAddress, res.Data)
}

func (p PrecompiledWasm) executeContract(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {
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

	return method.Outputs.Pack(res.Data)
}

func (p PrecompiledWasm) migrateContract(ctx sdk.Context, sender common.Address, method *abi.Method, args []interface{}) ([]byte, error) {

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
