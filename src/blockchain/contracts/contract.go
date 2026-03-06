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
)

// ContractMetaData contains all meta data concerning the Contract contract.
var ContractMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_keyName\",\"type\":\"bytes32\"}],\"name\":\"getIPNSKey\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"\",\"type\":\"bytes\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bytes32\",\"name\":\"_keyName\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_hash1\",\"type\":\"bytes32\"},{\"internalType\":\"bytes32\",\"name\":\"_hash2\",\"type\":\"bytes32\"}],\"name\":\"setIPNSKey\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506101d8806100206000396000f3fe608060405234801561001057600080fd5b50600436106100365760003560e01c8063cf13903a1461003b578063e95a84ac14610065575b600080fd5b61004e6100493660046100ff565b6100a2565b60405161005c929190610118565b60405180910390f35b6100a0610073366004610176565b60009283526020839052604090922080546001600160a01b03191633178155600181019190915560020155565b005b60008181526020819052604080822080546001820154600290920154835184815260608181018652946001600160a01b03909316939286919060208201818036833750505060208101939093525060408201529094909350915050565b60006020828403121561011157600080fd5b5035919050565b60018060a01b038316815260006020604081840152835180604085015260005b8181101561015457858101830151858201606001528201610138565b506000606082860101526060601f19601f830116850101925050509392505050565b60008060006060848603121561018b57600080fd5b50508135936020830135935060409092013591905056fea26469706673582212206829fd1c80099a717675ae95928f23db4376eace9d1a9b67a0019dc0200167f964736f6c63430008130033",
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
	parsed, err := abi.JSON(strings.NewReader(ContractABI))
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, parsed, caller, transactor, filterer), nil
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

// GetIPNSKey is a free data retrieval call binding the contract method 0xcf13903a.
//
// Solidity: function getIPNSKey(bytes32 _keyName) view returns(address, bytes)
func (_Contract *ContractCaller) GetIPNSKey(opts *bind.CallOpts, _keyName [32]byte) (common.Address, []byte, error) {
	var out []interface{}
	err := _Contract.contract.Call(opts, &out, "getIPNSKey", _keyName)

	if err != nil {
		return *new(common.Address), *new([]byte), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)
	out1 := *abi.ConvertType(out[1], new([]byte)).(*[]byte)

	return out0, out1, err

}

// GetIPNSKey is a free data retrieval call binding the contract method 0xcf13903a.
//
// Solidity: function getIPNSKey(bytes32 _keyName) view returns(address, bytes)
func (_Contract *ContractSession) GetIPNSKey(_keyName [32]byte) (common.Address, []byte, error) {
	return _Contract.Contract.GetIPNSKey(&_Contract.CallOpts, _keyName)
}

// GetIPNSKey is a free data retrieval call binding the contract method 0xcf13903a.
//
// Solidity: function getIPNSKey(bytes32 _keyName) view returns(address, bytes)
func (_Contract *ContractCallerSession) GetIPNSKey(_keyName [32]byte) (common.Address, []byte, error) {
	return _Contract.Contract.GetIPNSKey(&_Contract.CallOpts, _keyName)
}

// SetIPNSKey is a paid mutator transaction binding the contract method 0xe95a84ac.
//
// Solidity: function setIPNSKey(bytes32 _keyName, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractTransactor) SetIPNSKey(opts *bind.TransactOpts, _keyName [32]byte, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.contract.Transact(opts, "setIPNSKey", _keyName, _hash1, _hash2)
}

// SetIPNSKey is a paid mutator transaction binding the contract method 0xe95a84ac.
//
// Solidity: function setIPNSKey(bytes32 _keyName, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractSession) SetIPNSKey(_keyName [32]byte, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetIPNSKey(&_Contract.TransactOpts, _keyName, _hash1, _hash2)
}

// SetIPNSKey is a paid mutator transaction binding the contract method 0xe95a84ac.
//
// Solidity: function setIPNSKey(bytes32 _keyName, bytes32 _hash1, bytes32 _hash2) returns()
func (_Contract *ContractTransactorSession) SetIPNSKey(_keyName [32]byte, _hash1 [32]byte, _hash2 [32]byte) (*types.Transaction, error) {
	return _Contract.Contract.SetIPNSKey(&_Contract.TransactOpts, _keyName, _hash1, _hash2)
}
