// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contract

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

// ContractMetaData contains all meta data concerning the Contract contract.
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_functionName\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_messageID\",\"type\":\"uint64\"}],\"name\":\"getDocument\",\"outputs\":[{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_functionName\",\"type\":\"bytes32\"},{\"internalType\":\"uint64\",\"name\":\"_messageID\",\"type\":\"uint64\"},{\"internalType\":\"bytes32\",\"name\":\"_hash1\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_hash2\",\"type\":\"bytes32\"}],\"name\":\"setDocument\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x6080604052348015600e575f5ffd5b506101e38061001c5f395ff3fe608060405234801561000f575f5ffd5b5060043610610034575f3560e01c806349b5353914610038578063801fc76514610061575b5f5ffd5b61004b610046366004610116565b6100a0565b6040516100589190610140565b60405180910390f35b61009e61006f366004610175565b5f9384526020848152604080862067ffffffffffffffff95909516865293905291909220918255600190910155565b005b5f8281526020818152604080832067ffffffffffffffff8516845282528083208054600190910154825183815260608082018552959294919390918201818036833750505060208101939093525060408201529392505050565b803567ffffffffffffffff81168114610111575f5ffd5b919050565b5f5f60408385031215610127575f5ffd5b82359150610137602084016100fa565b90509250929050565b602081525f82518060208401528060208501604085015e5f604082850101526040601f19601f83011684010191505092915050565b5f5f5f5f60808587031215610188575f5ffd5b84359350610198602086016100fa565b9396939550505050604082013591606001359056fea2646970667358221220f90772a916299e61956fc67ebb0c5d752f654a870030b0bfa716df629c2af18e64736f6c634300081c0033",
}

// ContractABI is the input ABI used to generate the binding from.
// Deprecated: Use ContractMetaData.ABI instead.
var ContractABI = ContractMetaData.ABI

// ContractBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use ContractMetaData.Bin instead.
var ContractBin = ContractMetaData.Bin

// DeployContract deploys a new Ethereum contract, binding an instance of Contract to it.
func DeployContract(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Contract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(ContractBin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// Contract is an auto generated Go binding around an Ethereum contract.
type Contract struct {
	ContractCaller     // Read-only binding to the contract
	ContractTransactor // Write-only binding to the contract
	ContractFilterer   // Log filterer for contract events
}

// ContractCaller is an auto generated read-only Go binding around an Ethereum contract.
type ContractCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractTransactor is an auto generated write-only Go binding around an Ethereum contract.
type ContractTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type ContractFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// ContractSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type ContractSession struct {
	Contract     *Contract         // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// ContractCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type ContractCallerSession struct {
	Contract *ContractCaller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts   // Call options to use throughout this session
}

// ContractTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type ContractTransactorSession struct {
	Contract     *ContractTransactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts   // Transaction auth options to use throughout this session
}

// ContractRaw is an auto generated low-level Go binding around an Ethereum contract.
type ContractRaw struct {
	Contract *Contract // Generic contract binding to access the raw methods on
}

// ContractCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type ContractCallerRaw struct {
	Contract *ContractCaller // Generic read-only contract binding to access the raw methods on
}

// ContractTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type ContractTransactorRaw struct {
	Contract *ContractTransactor // Generic write-only contract binding to access the raw methods on
}

// NewContract creates a new instance of Contract, bound to a specific deployed contract.
func NewContract(address common.Address, backend bind.ContractBackend) (*Contract, error) {
	contract, err := bindContract(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Contract{ContractCaller: ContractCaller{contract: contract}, ContractTransactor: ContractTransactor{contract: contract}, ContractFilterer: ContractFilterer{contract: contract}}, nil
}

// NewContractCaller creates a new read-only instance of Contract, bound to a specific deployed contract.
func NewContractCaller(address common.Address, caller bind.ContractCaller) (*ContractCaller, error) {
	contract, err := bindContract(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &ContractCaller{contract: contract}, nil
}

// NewContractTransactor creates a new write-only instance of Contract, bound to a specific deployed contract.
func NewContractTransactor(address common.Address, transactor bind.ContractTransactor) (*ContractTransactor, error) {
	contract, err := bindContract(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &ContractTransactor{contract: contract}, nil
}

// NewContractFilterer creates a new log filterer instance of Contract, bound to a specific deployed contract.
func NewContractFilterer(address common.Address, filterer bind.ContractFilterer) (*ContractFilterer, error) {
	contract, err := bindContract(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &ContractFilterer{contract: contract}, nil
}

// bindContract binds a generic wrapper to an already deployed contract.
func bindContract(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.ContractCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.ContractTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Contract *ContractCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Contract.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Contract *ContractTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Contract *ContractTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Contract.Contract.contract.Transact(opts, method, params...)
}

// GetDocument is a free data retrieval call binding the contract method 0x49b53539.
//
// Solidity: function getDocument(bytes32 _functionName, uint64 _messageID) view returns(bytes)
func (_Contract *ContractCaller) GetDocument(opts *bind.CallOpts, _functionName [32]byte, _messageID uint64) ([]byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getDocument", _functionName, _messageID)

	if err != nil {
		return *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([]byte)).(*[]byte)

	return out0, err

}

// GetDocument is a free data retrieval call binding the contract method 0x49b53539.
//
// Solidity: function getDocument(bytes32 _functionName, uint64 _messageID) view returns(bytes)
func (_Contract *ContractSession) GetDocument(_functionName [32]byte, _messageID uint64) ([]byte, error) {
	return _Contract.Contract.GetDocument(&_Contract.CallOpts, _functionName, _messageID)
}

// GetDocument is a free data retrieval call binding the contract method 0x49b53539.
//
// Solidity: function getDocument(bytes32 _functionName, uint64 _messageID) view returns(bytes)
func (_Contract *ContractCallerSession) GetDocument(_functionName [32]byte, _messageID uint64) ([]byte, error) {
	return _Contract.Contract.GetDocument(&_Contract.CallOpts, _functionName, _messageID)
}

// SetDocument is a paid mutator transaction binding the contract method 0x801fc765.
//
// Solidity: function setDocument(bytes32 _functionName, uint64 _messageID, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractTransactor) SetDocument(opts *bind.TransactOpts, _functionName [32]byte, _messageID uint64, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setDocument", _functionName, _messageID, _hash1, _hash2)
}

// SetDocument is a paid mutator transaction binding the contract method 0x801fc765.
//
// Solidity: function setDocument(bytes32 _functionName, uint64 _messageID, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractSession) SetDocument(_functionName [32]byte, _messageID uint64, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetDocument(&_Contract.TransactOpts, _functionName, _messageID, _hash1, _hash2)
}

// SetDocument is a paid mutator transaction binding the contract method 0x801fc765.
//
// Solidity: function setDocument(bytes32 _functionName, uint64 _messageID, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractTransactorSession) SetDocument(_functionName [32]byte, _messageID uint64, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetDocument(&_Contract.TransactOpts, _functionName, _messageID, _hash1, _hash2)
}
