// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package e2e

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// NestedTransferInterfaceMetaData contains all meta data concerning the NestedTransferInterface contract.
var NestedTransferInterfaceMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_token\",\"type\":\"address\"},{\"internalType\":\"string\",\"name\":\"_cw20ContractAddress\",\"type\":\"string\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"value\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"length\",\"type\":\"uint256\"}],\"name\":\"StringsInsufficientHexLength\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint112\",\"name\":\"value\",\"type\":\"uint112\"}],\"name\":\"executeBankTransfer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"to\",\"type\":\"address\"},{\"internalType\":\"uint112\",\"name\":\"value\",\"type\":\"uint112\"}],\"name\":\"executeTransfer\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
}

// NestedTransferInterfaceABI is the input ABI used to generate the binding from.
// Deprecated: Use NestedTransferInterfaceMetaData.ABI instead.
var NestedTransferInterfaceABI = NestedTransferInterfaceMetaData.ABI

// NestedTransferInterface is an auto generated Go binding around an Ethereum contract.
type NestedTransferInterface struct {
	NestedTransferInterfaceCaller     // Read-only binding to the contract
	NestedTransferInterfaceTransactor // Write-only binding to the contract
	NestedTransferInterfaceFilterer   // Log filterer for contract events
}

// NestedTransferInterfaceCaller is an auto generated read-only Go binding around an Ethereum contract.
type NestedTransferInterfaceCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NestedTransferInterfaceTransactor is an auto generated write-only Go binding around an Ethereum contract.
type NestedTransferInterfaceTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NestedTransferInterfaceFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type NestedTransferInterfaceFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// NestedTransferInterfaceSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type NestedTransferInterfaceSession struct {
	Contract     *NestedTransferInterface // Generic contract binding to set the session for
	CallOpts     bind.CallOpts            // Call options to use throughout this session
	TransactOpts bind.TransactOpts        // Transaction auth options to use throughout this session
}

// NestedTransferInterfaceCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type NestedTransferInterfaceCallerSession struct {
	Contract *NestedTransferInterfaceCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts                  // Call options to use throughout this session
}

// NestedTransferInterfaceTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type NestedTransferInterfaceTransactorSession struct {
	Contract     *NestedTransferInterfaceTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts                  // Transaction auth options to use throughout this session
}

// NestedTransferInterfaceRaw is an auto generated low-level Go binding around an Ethereum contract.
type NestedTransferInterfaceRaw struct {
	Contract *NestedTransferInterface // Generic contract binding to access the raw methods on
}

// NestedTransferInterfaceCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type NestedTransferInterfaceCallerRaw struct {
	Contract *NestedTransferInterfaceCaller // Generic read-only contract binding to access the raw methods on
}

// NestedTransferInterfaceTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type NestedTransferInterfaceTransactorRaw struct {
	Contract *NestedTransferInterfaceTransactor // Generic write-only contract binding to access the raw methods on
}

// NewNestedTransferInterface creates a new instance of NestedTransferInterface, bound to a specific deployed contract.
func NewNestedTransferInterface(address common.Address, backend bind.ContractBackend) (*NestedTransferInterface, error) {
	contract, err := bindNestedTransferInterface(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &NestedTransferInterface{NestedTransferInterfaceCaller: NestedTransferInterfaceCaller{contract: contract}, NestedTransferInterfaceTransactor: NestedTransferInterfaceTransactor{contract: contract}, NestedTransferInterfaceFilterer: NestedTransferInterfaceFilterer{contract: contract}}, nil
}

// NewNestedTransferInterfaceCaller creates a new read-only instance of NestedTransferInterface, bound to a specific deployed contract.
func NewNestedTransferInterfaceCaller(address common.Address, caller bind.ContractCaller) (*NestedTransferInterfaceCaller, error) {
	contract, err := bindNestedTransferInterface(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &NestedTransferInterfaceCaller{contract: contract}, nil
}

// NewNestedTransferInterfaceTransactor creates a new write-only instance of NestedTransferInterface, bound to a specific deployed contract.
func NewNestedTransferInterfaceTransactor(address common.Address, transactor bind.ContractTransactor) (*NestedTransferInterfaceTransactor, error) {
	contract, err := bindNestedTransferInterface(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &NestedTransferInterfaceTransactor{contract: contract}, nil
}

// NewNestedTransferInterfaceFilterer creates a new log filterer instance of NestedTransferInterface, bound to a specific deployed contract.
func NewNestedTransferInterfaceFilterer(address common.Address, filterer bind.ContractFilterer) (*NestedTransferInterfaceFilterer, error) {
	contract, err := bindNestedTransferInterface(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &NestedTransferInterfaceFilterer{contract: contract}, nil
}

// bindNestedTransferInterface binds a generic wrapper to an already deployed contract.
func bindNestedTransferInterface(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := NestedTransferInterfaceMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NestedTransferInterface *NestedTransferInterfaceRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NestedTransferInterface.Contract.NestedTransferInterfaceCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NestedTransferInterface *NestedTransferInterfaceRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.NestedTransferInterfaceTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NestedTransferInterface *NestedTransferInterfaceRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.NestedTransferInterfaceTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_NestedTransferInterface *NestedTransferInterfaceCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _NestedTransferInterface.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_NestedTransferInterface *NestedTransferInterfaceTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_NestedTransferInterface *NestedTransferInterfaceTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.contract.Transact(opts, method, params...)
}

// ExecuteBankTransfer is a paid mutator transaction binding the contract method 0xa19fb37d.
//
// Solidity: function executeBankTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceTransactor) ExecuteBankTransfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.contract.Transact(opts, "executeBankTransfer", to, value)
}

// ExecuteBankTransfer is a paid mutator transaction binding the contract method 0xa19fb37d.
//
// Solidity: function executeBankTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceSession) ExecuteBankTransfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.ExecuteBankTransfer(&_NestedTransferInterface.TransactOpts, to, value)
}

// ExecuteBankTransfer is a paid mutator transaction binding the contract method 0xa19fb37d.
//
// Solidity: function executeBankTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceTransactorSession) ExecuteBankTransfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.ExecuteBankTransfer(&_NestedTransferInterface.TransactOpts, to, value)
}

// ExecuteTransfer is a paid mutator transaction binding the contract method 0x8eb7a66a.
//
// Solidity: function executeTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceTransactor) ExecuteTransfer(opts *bind.TransactOpts, to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.contract.Transact(opts, "executeTransfer", to, value)
}

// ExecuteTransfer is a paid mutator transaction binding the contract method 0x8eb7a66a.
//
// Solidity: function executeTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceSession) ExecuteTransfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.ExecuteTransfer(&_NestedTransferInterface.TransactOpts, to, value)
}

// ExecuteTransfer is a paid mutator transaction binding the contract method 0x8eb7a66a.
//
// Solidity: function executeTransfer(address to, uint112 value) returns()
func (_NestedTransferInterface *NestedTransferInterfaceTransactorSession) ExecuteTransfer(to common.Address, value *big.Int) (*types.Transaction, error) {
	return _NestedTransferInterface.Contract.ExecuteTransfer(&_NestedTransferInterface.TransactOpts, to, value)
}
