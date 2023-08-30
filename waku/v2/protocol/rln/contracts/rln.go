// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package contracts

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

// RLNMetaData contains all meta data concerning the RLN contract.
var RLNMetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"internalType\":\"address\",\"name\":\"_poseidonHasher\",\"type\":\"address\"},{\"internalType\":\"uint16\",\"name\":\"_contractIndex\",\"type\":\"uint16\"}],\"stateMutability\":\"nonpayable\",\"type\":\"constructor\"},{\"inputs\":[],\"name\":\"DuplicateIdCommitment\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"FullTree\",\"type\":\"error\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"InvalidIdCommitment\",\"type\":\"error\"},{\"inputs\":[],\"name\":\"NotImplemented\",\"type\":\"error\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"MemberRegistered\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"},{\"indexed\":false,\"internalType\":\"uint256\",\"name\":\"index\",\"type\":\"uint256\"}],\"name\":\"MemberWithdrawn\",\"type\":\"event\"},{\"anonymous\":false,\"inputs\":[{\"indexed\":true,\"internalType\":\"address\",\"name\":\"previousOwner\",\"type\":\"address\"},{\"indexed\":true,\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"OwnershipTransferred\",\"type\":\"event\"},{\"inputs\":[],\"name\":\"DEPTH\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"MEMBERSHIP_DEPOSIT\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"SET_SIZE\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"contractIndex\",\"outputs\":[{\"internalType\":\"uint16\",\"name\":\"\",\"type\":\"uint16\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"deployedBlockNumber\",\"outputs\":[{\"internalType\":\"uint32\",\"name\":\"\",\"type\":\"uint32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"idCommitmentIndex\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"isValidCommitment\",\"outputs\":[{\"internalType\":\"bool\",\"name\":\"\",\"type\":\"bool\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"members\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"owner\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"poseidonHasher\",\"outputs\":[{\"internalType\":\"contractPoseidonHasher\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256[]\",\"name\":\"idCommitments\",\"type\":\"uint256[]\"}],\"name\":\"register\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"}],\"name\":\"register\",\"outputs\":[],\"stateMutability\":\"payable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"renounceOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"idCommitment\",\"type\":\"uint256\"},{\"internalType\":\"addresspayable\",\"name\":\"receiver\",\"type\":\"address\"},{\"internalType\":\"uint256[8]\",\"name\":\"proof\",\"type\":\"uint256[8]\"}],\"name\":\"slash\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"name\":\"stakedAmounts\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"newOwner\",\"type\":\"address\"}],\"name\":\"transferOwnership\",\"outputs\":[],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"verifier\",\"outputs\":[{\"internalType\":\"contractIVerifier\",\"name\":\"\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"withdraw\",\"outputs\":[],\"stateMutability\":\"pure\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"\",\"type\":\"address\"}],\"name\":\"withdrawalBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
	Bin: "0x610160604052600180553480156200001657600080fd5b50604051620014533803806200145383398181016040528101906200003c91906200028f565b6000601483600062000063620000576200011a60201b60201c565b6200012260201b60201c565b83608081815250508260a08181525050826001901b60c081815250508173ffffffffffffffffffffffffffffffffffffffff1660e08173ffffffffffffffffffffffffffffffffffffffff16815250508073ffffffffffffffffffffffffffffffffffffffff166101008173ffffffffffffffffffffffffffffffffffffffff16815250504363ffffffff166101208163ffffffff1681525050505050508061ffff166101408161ffff16815250505050620002d6565b600033905090565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006200021882620001eb565b9050919050565b6200022a816200020b565b81146200023657600080fd5b50565b6000815190506200024a816200021f565b92915050565b600061ffff82169050919050565b620002698162000250565b81146200027557600080fd5b50565b60008151905062000289816200025e565b92915050565b60008060408385031215620002a957620002a8620001e6565b5b6000620002b98582860162000239565b9250506020620002cc8582860162000278565b9150509250929050565b60805160a05160c05160e0516101005161012051610140516111146200033f60003960006104fd0152600061059b0152600061052101526000818161046401526105450152600081816106ee0152610a44015260006106940152600061074401526111146000f3fe6080604052600436106101145760003560e01c80638be9b119116100a0578063c5b208ff11610064578063c5b208ff1461037d578063d0383d68146103ba578063f207564e146103e5578063f220b9ec14610401578063f2fde38b1461042c57610114565b80638be9b119146102965780638da5cb5b146102bf57806398366e35146102ea578063ae74552a14610315578063bc4991281461034057610114565b80633ccfd60b116100e75780633ccfd60b146101d75780634add651e146101ee5780635daf08ca14610219578063715018a6146102565780637a34289d1461026d57610114565b806322d9730c1461011957806328b070e0146101565780632b7ac3f314610181578063331b6ab3146101ac575b600080fd5b34801561012557600080fd5b50610140600480360381019061013b9190610ae0565b610455565b60405161014d9190610b28565b60405180910390f35b34801561016257600080fd5b5061016b6104fb565b6040516101789190610b60565b60405180910390f35b34801561018d57600080fd5b5061019661051f565b6040516101a39190610bfa565b60405180910390f35b3480156101b857600080fd5b506101c1610543565b6040516101ce9190610c36565b60405180910390f35b3480156101e357600080fd5b506101ec610567565b005b3480156101fa57600080fd5b50610203610599565b6040516102109190610c70565b60405180910390f35b34801561022557600080fd5b50610240600480360381019061023b9190610ae0565b6105bd565b60405161024d9190610c9a565b60405180910390f35b34801561026257600080fd5b5061026b6105d5565b005b34801561027957600080fd5b50610294600480360381019061028f9190610d1a565b6105e9565b005b3480156102a257600080fd5b506102bd60048036038101906102b89190610dc7565b610637565b005b3480156102cb57600080fd5b506102d4610669565b6040516102e19190610e3c565b60405180910390f35b3480156102f657600080fd5b506102ff610692565b60405161030c9190610c9a565b60405180910390f35b34801561032157600080fd5b5061032a6106b6565b6040516103379190610c9a565b60405180910390f35b34801561034c57600080fd5b5061036760048036038101906103629190610ae0565b6106bc565b6040516103749190610c9a565b60405180910390f35b34801561038957600080fd5b506103a4600480360381019061039f9190610e83565b6106d4565b6040516103b19190610c9a565b60405180910390f35b3480156103c657600080fd5b506103cf6106ec565b6040516103dc9190610c9a565b60405180910390f35b6103ff60048036038101906103fa9190610ae0565b610710565b005b34801561040d57600080fd5b50610416610742565b6040516104239190610c9a565b60405180910390f35b34801561043857600080fd5b50610453600480360381019061044e9190610e83565b610766565b005b60008082141580156104f457507f000000000000000000000000000000000000000000000000000000000000000073ffffffffffffffffffffffffffffffffffffffff1663e493ef8c6040518163ffffffff1660e01b8152600401602060405180830381865afa1580156104cd573d6000803e3d6000fd5b505050506040513d601f19601f820116820180604052508101906104f19190610ec5565b82105b9050919050565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000081565b7f000000000000000000000000000000000000000000000000000000000000000081565b6040517fd623472500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000081565b60036020528060005260406000206000915090505481565b6105dd6107e9565b6105e76000610867565b565b6105f16107e9565b600082829050905060005b818110156106315761062684848381811061061a57610619610ef2565b5b9050602002013561092b565b8060010190506105fc565b50505050565b6040517fd623472500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff16905090565b7f000000000000000000000000000000000000000000000000000000000000000081565b60015481565b60026020528060005260406000206000915090505481565b60046020528060005260406000206000915090505481565b7f000000000000000000000000000000000000000000000000000000000000000081565b6040517fd623472500000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000081565b61076e6107e9565b600073ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff16036107dd576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016107d490610fa4565b60405180910390fd5b6107e681610867565b50565b6107f16109a4565b73ffffffffffffffffffffffffffffffffffffffff1661080f610669565b73ffffffffffffffffffffffffffffffffffffffff1614610865576040517f08c379a000000000000000000000000000000000000000000000000000000000815260040161085c90611010565b60405180910390fd5b565b60008060009054906101000a900473ffffffffffffffffffffffffffffffffffffffff169050816000806101000a81548173ffffffffffffffffffffffffffffffffffffffff021916908373ffffffffffffffffffffffffffffffffffffffff1602179055508173ffffffffffffffffffffffffffffffffffffffff168173ffffffffffffffffffffffffffffffffffffffff167f8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e060405160405180910390a35050565b610934816109ac565b600160036000838152602001908152602001600020819055507f5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d281600154604051610980929190611030565b60405180910390a1600180600082825461099a9190611088565b9250508190555050565b600033905090565b6109b581610455565b6109f657806040517f7f3e75af0000000000000000000000000000000000000000000000000000000081526004016109ed9190610c9a565b60405180910390fd5b6000600360008381526020019081526020016000205414610a42576040517e0a60f700000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b7f000000000000000000000000000000000000000000000000000000000000000060015410610a9d576040517f57f6953100000000000000000000000000000000000000000000000000000000815260040160405180910390fd5b50565b600080fd5b600080fd5b6000819050919050565b610abd81610aaa565b8114610ac857600080fd5b50565b600081359050610ada81610ab4565b92915050565b600060208284031215610af657610af5610aa0565b5b6000610b0484828501610acb565b91505092915050565b60008115159050919050565b610b2281610b0d565b82525050565b6000602082019050610b3d6000830184610b19565b92915050565b600061ffff82169050919050565b610b5a81610b43565b82525050565b6000602082019050610b756000830184610b51565b92915050565b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b6000819050919050565b6000610bc0610bbb610bb684610b7b565b610b9b565b610b7b565b9050919050565b6000610bd282610ba5565b9050919050565b6000610be482610bc7565b9050919050565b610bf481610bd9565b82525050565b6000602082019050610c0f6000830184610beb565b92915050565b6000610c2082610bc7565b9050919050565b610c3081610c15565b82525050565b6000602082019050610c4b6000830184610c27565b92915050565b600063ffffffff82169050919050565b610c6a81610c51565b82525050565b6000602082019050610c856000830184610c61565b92915050565b610c9481610aaa565b82525050565b6000602082019050610caf6000830184610c8b565b92915050565b600080fd5b600080fd5b600080fd5b60008083601f840112610cda57610cd9610cb5565b5b8235905067ffffffffffffffff811115610cf757610cf6610cba565b5b602083019150836020820283011115610d1357610d12610cbf565b5b9250929050565b60008060208385031215610d3157610d30610aa0565b5b600083013567ffffffffffffffff811115610d4f57610d4e610aa5565b5b610d5b85828601610cc4565b92509250509250929050565b6000610d7282610b7b565b9050919050565b610d8281610d67565b8114610d8d57600080fd5b50565b600081359050610d9f81610d79565b92915050565b600081905082602060080282011115610dc157610dc0610cbf565b5b92915050565b60008060006101408486031215610de157610de0610aa0565b5b6000610def86828701610acb565b9350506020610e0086828701610d90565b9250506040610e1186828701610da5565b9150509250925092565b6000610e2682610b7b565b9050919050565b610e3681610e1b565b82525050565b6000602082019050610e516000830184610e2d565b92915050565b610e6081610e1b565b8114610e6b57600080fd5b50565b600081359050610e7d81610e57565b92915050565b600060208284031215610e9957610e98610aa0565b5b6000610ea784828501610e6e565b91505092915050565b600081519050610ebf81610ab4565b92915050565b600060208284031215610edb57610eda610aa0565b5b6000610ee984828501610eb0565b91505092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600082825260208201905092915050565b7f4f776e61626c653a206e6577206f776e657220697320746865207a65726f206160008201527f6464726573730000000000000000000000000000000000000000000000000000602082015250565b6000610f8e602683610f21565b9150610f9982610f32565b604082019050919050565b60006020820190508181036000830152610fbd81610f81565b9050919050565b7f4f776e61626c653a2063616c6c6572206973206e6f7420746865206f776e6572600082015250565b6000610ffa602083610f21565b915061100582610fc4565b602082019050919050565b6000602082019050818103600083015261102981610fed565b9050919050565b60006040820190506110456000830185610c8b565b6110526020830184610c8b565b9392505050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b600061109382610aaa565b915061109e83610aaa565b9250827fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff038211156110d3576110d2611059565b5b82820190509291505056fea2646970667358221220418f43f842c2bfcb16498620d0ec8af5bf2c69e3f7aa005ef11baa07a9c199c864736f6c634300080f0033",
}

// RLNABI is the input ABI used to generate the binding from.
// Deprecated: Use RLNMetaData.ABI instead.
var RLNABI = RLNMetaData.ABI

// RLNBin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use RLNMetaData.Bin instead.
var RLNBin = RLNMetaData.Bin

// DeployRLN deploys a new Ethereum contract, binding an instance of RLN to it.
func DeployRLN(auth *bind.TransactOpts, backend bind.ContractBackend, _poseidonHasher common.Address, _contractIndex uint16) (common.Address, *types.Transaction, *RLN, error) {
	parsed, err := RLNMetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(RLNBin), backend, _poseidonHasher, _contractIndex)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &RLN{RLNCaller: RLNCaller{contract: contract}, RLNTransactor: RLNTransactor{contract: contract}, RLNFilterer: RLNFilterer{contract: contract}}, nil
}

// RLN is an auto generated Go binding around an Ethereum contract.
type RLN struct {
	RLNCaller     // Read-only binding to the contract
	RLNTransactor // Write-only binding to the contract
	RLNFilterer   // Log filterer for contract events
}

// RLNCaller is an auto generated read-only Go binding around an Ethereum contract.
type RLNCaller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNTransactor is an auto generated write-only Go binding around an Ethereum contract.
type RLNTransactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNFilterer is an auto generated log filtering Go binding around an Ethereum contract events.
type RLNFilterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// RLNSession is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type RLNSession struct {
	Contract     *RLN              // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLNCallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type RLNCallerSession struct {
	Contract *RLNCaller    // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts // Call options to use throughout this session
}

// RLNTransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type RLNTransactorSession struct {
	Contract     *RLNTransactor    // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// RLNRaw is an auto generated low-level Go binding around an Ethereum contract.
type RLNRaw struct {
	Contract *RLN // Generic contract binding to access the raw methods on
}

// RLNCallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type RLNCallerRaw struct {
	Contract *RLNCaller // Generic read-only contract binding to access the raw methods on
}

// RLNTransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type RLNTransactorRaw struct {
	Contract *RLNTransactor // Generic write-only contract binding to access the raw methods on
}

// NewRLN creates a new instance of RLN, bound to a specific deployed contract.
func NewRLN(address common.Address, backend bind.ContractBackend) (*RLN, error) {
	contract, err := bindRLN(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &RLN{RLNCaller: RLNCaller{contract: contract}, RLNTransactor: RLNTransactor{contract: contract}, RLNFilterer: RLNFilterer{contract: contract}}, nil
}

// NewRLNCaller creates a new read-only instance of RLN, bound to a specific deployed contract.
func NewRLNCaller(address common.Address, caller bind.ContractCaller) (*RLNCaller, error) {
	contract, err := bindRLN(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &RLNCaller{contract: contract}, nil
}

// NewRLNTransactor creates a new write-only instance of RLN, bound to a specific deployed contract.
func NewRLNTransactor(address common.Address, transactor bind.ContractTransactor) (*RLNTransactor, error) {
	contract, err := bindRLN(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &RLNTransactor{contract: contract}, nil
}

// NewRLNFilterer creates a new log filterer instance of RLN, bound to a specific deployed contract.
func NewRLNFilterer(address common.Address, filterer bind.ContractFilterer) (*RLNFilterer, error) {
	contract, err := bindRLN(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &RLNFilterer{contract: contract}, nil
}

// bindRLN binds a generic wrapper to an already deployed contract.
func bindRLN(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := RLNMetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLN *RLNRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RLN.Contract.RLNCaller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLN *RLNRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLN.Contract.RLNTransactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLN *RLNRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLN.Contract.RLNTransactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_RLN *RLNCallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _RLN.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_RLN *RLNTransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLN.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_RLN *RLNTransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _RLN.Contract.contract.Transact(opts, method, params...)
}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNCaller) DEPTH(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "DEPTH")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNSession) DEPTH() (*big.Int, error) {
	return _RLN.Contract.DEPTH(&_RLN.CallOpts)
}

// DEPTH is a free data retrieval call binding the contract method 0x98366e35.
//
// Solidity: function DEPTH() view returns(uint256)
func (_RLN *RLNCallerSession) DEPTH() (*big.Int, error) {
	return _RLN.Contract.DEPTH(&_RLN.CallOpts)
}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNCaller) MEMBERSHIPDEPOSIT(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "MEMBERSHIP_DEPOSIT")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNSession) MEMBERSHIPDEPOSIT() (*big.Int, error) {
	return _RLN.Contract.MEMBERSHIPDEPOSIT(&_RLN.CallOpts)
}

// MEMBERSHIPDEPOSIT is a free data retrieval call binding the contract method 0xf220b9ec.
//
// Solidity: function MEMBERSHIP_DEPOSIT() view returns(uint256)
func (_RLN *RLNCallerSession) MEMBERSHIPDEPOSIT() (*big.Int, error) {
	return _RLN.Contract.MEMBERSHIPDEPOSIT(&_RLN.CallOpts)
}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNCaller) SETSIZE(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "SET_SIZE")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNSession) SETSIZE() (*big.Int, error) {
	return _RLN.Contract.SETSIZE(&_RLN.CallOpts)
}

// SETSIZE is a free data retrieval call binding the contract method 0xd0383d68.
//
// Solidity: function SET_SIZE() view returns(uint256)
func (_RLN *RLNCallerSession) SETSIZE() (*big.Int, error) {
	return _RLN.Contract.SETSIZE(&_RLN.CallOpts)
}

// ContractIndex is a free data retrieval call binding the contract method 0x28b070e0.
//
// Solidity: function contractIndex() view returns(uint16)
func (_RLN *RLNCaller) ContractIndex(opts *bind.CallOpts) (uint16, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "contractIndex")

	if err != nil {
		return *new(uint16), err
	}

	out0 := *abi.ConvertType(out[0], new(uint16)).(*uint16)

	return out0, err

}

// ContractIndex is a free data retrieval call binding the contract method 0x28b070e0.
//
// Solidity: function contractIndex() view returns(uint16)
func (_RLN *RLNSession) ContractIndex() (uint16, error) {
	return _RLN.Contract.ContractIndex(&_RLN.CallOpts)
}

// ContractIndex is a free data retrieval call binding the contract method 0x28b070e0.
//
// Solidity: function contractIndex() view returns(uint16)
func (_RLN *RLNCallerSession) ContractIndex() (uint16, error) {
	return _RLN.Contract.ContractIndex(&_RLN.CallOpts)
}

// DeployedBlockNumber is a free data retrieval call binding the contract method 0x4add651e.
//
// Solidity: function deployedBlockNumber() view returns(uint32)
func (_RLN *RLNCaller) DeployedBlockNumber(opts *bind.CallOpts) (uint32, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "deployedBlockNumber")

	if err != nil {
		return *new(uint32), err
	}

	out0 := *abi.ConvertType(out[0], new(uint32)).(*uint32)

	return out0, err

}

// DeployedBlockNumber is a free data retrieval call binding the contract method 0x4add651e.
//
// Solidity: function deployedBlockNumber() view returns(uint32)
func (_RLN *RLNSession) DeployedBlockNumber() (uint32, error) {
	return _RLN.Contract.DeployedBlockNumber(&_RLN.CallOpts)
}

// DeployedBlockNumber is a free data retrieval call binding the contract method 0x4add651e.
//
// Solidity: function deployedBlockNumber() view returns(uint32)
func (_RLN *RLNCallerSession) DeployedBlockNumber() (uint32, error) {
	return _RLN.Contract.DeployedBlockNumber(&_RLN.CallOpts)
}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNCaller) IdCommitmentIndex(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "idCommitmentIndex")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNSession) IdCommitmentIndex() (*big.Int, error) {
	return _RLN.Contract.IdCommitmentIndex(&_RLN.CallOpts)
}

// IdCommitmentIndex is a free data retrieval call binding the contract method 0xae74552a.
//
// Solidity: function idCommitmentIndex() view returns(uint256)
func (_RLN *RLNCallerSession) IdCommitmentIndex() (*big.Int, error) {
	return _RLN.Contract.IdCommitmentIndex(&_RLN.CallOpts)
}

// IsValidCommitment is a free data retrieval call binding the contract method 0x22d9730c.
//
// Solidity: function isValidCommitment(uint256 idCommitment) view returns(bool)
func (_RLN *RLNCaller) IsValidCommitment(opts *bind.CallOpts, idCommitment *big.Int) (bool, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "isValidCommitment", idCommitment)

	if err != nil {
		return *new(bool), err
	}

	out0 := *abi.ConvertType(out[0], new(bool)).(*bool)

	return out0, err

}

// IsValidCommitment is a free data retrieval call binding the contract method 0x22d9730c.
//
// Solidity: function isValidCommitment(uint256 idCommitment) view returns(bool)
func (_RLN *RLNSession) IsValidCommitment(idCommitment *big.Int) (bool, error) {
	return _RLN.Contract.IsValidCommitment(&_RLN.CallOpts, idCommitment)
}

// IsValidCommitment is a free data retrieval call binding the contract method 0x22d9730c.
//
// Solidity: function isValidCommitment(uint256 idCommitment) view returns(bool)
func (_RLN *RLNCallerSession) IsValidCommitment(idCommitment *big.Int) (bool, error) {
	return _RLN.Contract.IsValidCommitment(&_RLN.CallOpts, idCommitment)
}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNCaller) Members(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "members", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNSession) Members(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.Members(&_RLN.CallOpts, arg0)
}

// Members is a free data retrieval call binding the contract method 0x5daf08ca.
//
// Solidity: function members(uint256 ) view returns(uint256)
func (_RLN *RLNCallerSession) Members(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.Members(&_RLN.CallOpts, arg0)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RLN *RLNCaller) Owner(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "owner")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RLN *RLNSession) Owner() (common.Address, error) {
	return _RLN.Contract.Owner(&_RLN.CallOpts)
}

// Owner is a free data retrieval call binding the contract method 0x8da5cb5b.
//
// Solidity: function owner() view returns(address)
func (_RLN *RLNCallerSession) Owner() (common.Address, error) {
	return _RLN.Contract.Owner(&_RLN.CallOpts)
}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNCaller) PoseidonHasher(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "poseidonHasher")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNSession) PoseidonHasher() (common.Address, error) {
	return _RLN.Contract.PoseidonHasher(&_RLN.CallOpts)
}

// PoseidonHasher is a free data retrieval call binding the contract method 0x331b6ab3.
//
// Solidity: function poseidonHasher() view returns(address)
func (_RLN *RLNCallerSession) PoseidonHasher() (common.Address, error) {
	return _RLN.Contract.PoseidonHasher(&_RLN.CallOpts)
}

// Slash is a free data retrieval call binding the contract method 0x8be9b119.
//
// Solidity: function slash(uint256 idCommitment, address receiver, uint256[8] proof) pure returns()
func (_RLN *RLNCaller) Slash(opts *bind.CallOpts, idCommitment *big.Int, receiver common.Address, proof [8]*big.Int) error {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "slash", idCommitment, receiver, proof)

	if err != nil {
		return err
	}

	return err

}

// Slash is a free data retrieval call binding the contract method 0x8be9b119.
//
// Solidity: function slash(uint256 idCommitment, address receiver, uint256[8] proof) pure returns()
func (_RLN *RLNSession) Slash(idCommitment *big.Int, receiver common.Address, proof [8]*big.Int) error {
	return _RLN.Contract.Slash(&_RLN.CallOpts, idCommitment, receiver, proof)
}

// Slash is a free data retrieval call binding the contract method 0x8be9b119.
//
// Solidity: function slash(uint256 idCommitment, address receiver, uint256[8] proof) pure returns()
func (_RLN *RLNCallerSession) Slash(idCommitment *big.Int, receiver common.Address, proof [8]*big.Int) error {
	return _RLN.Contract.Slash(&_RLN.CallOpts, idCommitment, receiver, proof)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNCaller) StakedAmounts(opts *bind.CallOpts, arg0 *big.Int) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "stakedAmounts", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNSession) StakedAmounts(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.StakedAmounts(&_RLN.CallOpts, arg0)
}

// StakedAmounts is a free data retrieval call binding the contract method 0xbc499128.
//
// Solidity: function stakedAmounts(uint256 ) view returns(uint256)
func (_RLN *RLNCallerSession) StakedAmounts(arg0 *big.Int) (*big.Int, error) {
	return _RLN.Contract.StakedAmounts(&_RLN.CallOpts, arg0)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_RLN *RLNCaller) Verifier(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "verifier")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_RLN *RLNSession) Verifier() (common.Address, error) {
	return _RLN.Contract.Verifier(&_RLN.CallOpts)
}

// Verifier is a free data retrieval call binding the contract method 0x2b7ac3f3.
//
// Solidity: function verifier() view returns(address)
func (_RLN *RLNCallerSession) Verifier() (common.Address, error) {
	return _RLN.Contract.Verifier(&_RLN.CallOpts)
}

// Withdraw is a free data retrieval call binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() pure returns()
func (_RLN *RLNCaller) Withdraw(opts *bind.CallOpts) error {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "withdraw")

	if err != nil {
		return err
	}

	return err

}

// Withdraw is a free data retrieval call binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() pure returns()
func (_RLN *RLNSession) Withdraw() error {
	return _RLN.Contract.Withdraw(&_RLN.CallOpts)
}

// Withdraw is a free data retrieval call binding the contract method 0x3ccfd60b.
//
// Solidity: function withdraw() pure returns()
func (_RLN *RLNCallerSession) Withdraw() error {
	return _RLN.Contract.Withdraw(&_RLN.CallOpts)
}

// WithdrawalBalance is a free data retrieval call binding the contract method 0xc5b208ff.
//
// Solidity: function withdrawalBalance(address ) view returns(uint256)
func (_RLN *RLNCaller) WithdrawalBalance(opts *bind.CallOpts, arg0 common.Address) (*big.Int, error) {
	var out []interface{}
	err := _RLN.contract.Call(opts, &out, "withdrawalBalance", arg0)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// WithdrawalBalance is a free data retrieval call binding the contract method 0xc5b208ff.
//
// Solidity: function withdrawalBalance(address ) view returns(uint256)
func (_RLN *RLNSession) WithdrawalBalance(arg0 common.Address) (*big.Int, error) {
	return _RLN.Contract.WithdrawalBalance(&_RLN.CallOpts, arg0)
}

// WithdrawalBalance is a free data retrieval call binding the contract method 0xc5b208ff.
//
// Solidity: function withdrawalBalance(address ) view returns(uint256)
func (_RLN *RLNCallerSession) WithdrawalBalance(arg0 common.Address) (*big.Int, error) {
	return _RLN.Contract.WithdrawalBalance(&_RLN.CallOpts, arg0)
}

// Register is a paid mutator transaction binding the contract method 0x7a34289d.
//
// Solidity: function register(uint256[] idCommitments) returns()
func (_RLN *RLNTransactor) Register(opts *bind.TransactOpts, idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "register", idCommitments)
}

// Register is a paid mutator transaction binding the contract method 0x7a34289d.
//
// Solidity: function register(uint256[] idCommitments) returns()
func (_RLN *RLNSession) Register(idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, idCommitments)
}

// Register is a paid mutator transaction binding the contract method 0x7a34289d.
//
// Solidity: function register(uint256[] idCommitments) returns()
func (_RLN *RLNTransactorSession) Register(idCommitments []*big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register(&_RLN.TransactOpts, idCommitments)
}

// Register0 is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNTransactor) Register0(opts *bind.TransactOpts, idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "register0", idCommitment)
}

// Register0 is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNSession) Register0(idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register0(&_RLN.TransactOpts, idCommitment)
}

// Register0 is a paid mutator transaction binding the contract method 0xf207564e.
//
// Solidity: function register(uint256 idCommitment) payable returns()
func (_RLN *RLNTransactorSession) Register0(idCommitment *big.Int) (*types.Transaction, error) {
	return _RLN.Contract.Register0(&_RLN.TransactOpts, idCommitment)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_RLN *RLNTransactor) RenounceOwnership(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "renounceOwnership")
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_RLN *RLNSession) RenounceOwnership() (*types.Transaction, error) {
	return _RLN.Contract.RenounceOwnership(&_RLN.TransactOpts)
}

// RenounceOwnership is a paid mutator transaction binding the contract method 0x715018a6.
//
// Solidity: function renounceOwnership() returns()
func (_RLN *RLNTransactorSession) RenounceOwnership() (*types.Transaction, error) {
	return _RLN.Contract.RenounceOwnership(&_RLN.TransactOpts)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_RLN *RLNTransactor) TransferOwnership(opts *bind.TransactOpts, newOwner common.Address) (*types.Transaction, error) {
	return _RLN.contract.Transact(opts, "transferOwnership", newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_RLN *RLNSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _RLN.Contract.TransferOwnership(&_RLN.TransactOpts, newOwner)
}

// TransferOwnership is a paid mutator transaction binding the contract method 0xf2fde38b.
//
// Solidity: function transferOwnership(address newOwner) returns()
func (_RLN *RLNTransactorSession) TransferOwnership(newOwner common.Address) (*types.Transaction, error) {
	return _RLN.Contract.TransferOwnership(&_RLN.TransactOpts, newOwner)
}

// RLNMemberRegisteredIterator is returned from FilterMemberRegistered and is used to iterate over the raw logs and unpacked data for MemberRegistered events raised by the RLN contract.
type RLNMemberRegisteredIterator struct {
	Event *RLNMemberRegistered // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RLNMemberRegisteredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RLNMemberRegistered)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RLNMemberRegistered)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RLNMemberRegisteredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RLNMemberRegisteredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RLNMemberRegistered represents a MemberRegistered event raised by the RLN contract.
type RLNMemberRegistered struct {
	IdCommitment *big.Int
	Index        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterMemberRegistered is a free log retrieval operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) FilterMemberRegistered(opts *bind.FilterOpts) (*RLNMemberRegisteredIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberRegistered")
	if err != nil {
		return nil, err
	}
	return &RLNMemberRegisteredIterator{contract: _RLN.contract, event: "MemberRegistered", logs: logs, sub: sub}, nil
}

// WatchMemberRegistered is a free log subscription operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) WatchMemberRegistered(opts *bind.WatchOpts, sink chan<- *RLNMemberRegistered) (event.Subscription, error) {

	logs, sub, err := _RLN.contract.WatchLogs(opts, "MemberRegistered")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RLNMemberRegistered)
				if err := _RLN.contract.UnpackLog(event, "MemberRegistered", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMemberRegistered is a log parse operation binding the contract event 0x5a92c2530f207992057b9c3e544108ffce3beda4a63719f316967c49bf6159d2.
//
// Solidity: event MemberRegistered(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) ParseMemberRegistered(log types.Log) (*RLNMemberRegistered, error) {
	event := new(RLNMemberRegistered)
	if err := _RLN.contract.UnpackLog(event, "MemberRegistered", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RLNMemberWithdrawnIterator is returned from FilterMemberWithdrawn and is used to iterate over the raw logs and unpacked data for MemberWithdrawn events raised by the RLN contract.
type RLNMemberWithdrawnIterator struct {
	Event *RLNMemberWithdrawn // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RLNMemberWithdrawnIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RLNMemberWithdrawn)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RLNMemberWithdrawn)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RLNMemberWithdrawnIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RLNMemberWithdrawnIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RLNMemberWithdrawn represents a MemberWithdrawn event raised by the RLN contract.
type RLNMemberWithdrawn struct {
	IdCommitment *big.Int
	Index        *big.Int
	Raw          types.Log // Blockchain specific contextual infos
}

// FilterMemberWithdrawn is a free log retrieval operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) FilterMemberWithdrawn(opts *bind.FilterOpts) (*RLNMemberWithdrawnIterator, error) {

	logs, sub, err := _RLN.contract.FilterLogs(opts, "MemberWithdrawn")
	if err != nil {
		return nil, err
	}
	return &RLNMemberWithdrawnIterator{contract: _RLN.contract, event: "MemberWithdrawn", logs: logs, sub: sub}, nil
}

// WatchMemberWithdrawn is a free log subscription operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) WatchMemberWithdrawn(opts *bind.WatchOpts, sink chan<- *RLNMemberWithdrawn) (event.Subscription, error) {

	logs, sub, err := _RLN.contract.WatchLogs(opts, "MemberWithdrawn")
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RLNMemberWithdrawn)
				if err := _RLN.contract.UnpackLog(event, "MemberWithdrawn", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseMemberWithdrawn is a log parse operation binding the contract event 0x62ec3a516d22a993ce5cb4e7593e878c74f4d799dde522a88dc27a994fd5a943.
//
// Solidity: event MemberWithdrawn(uint256 idCommitment, uint256 index)
func (_RLN *RLNFilterer) ParseMemberWithdrawn(log types.Log) (*RLNMemberWithdrawn, error) {
	event := new(RLNMemberWithdrawn)
	if err := _RLN.contract.UnpackLog(event, "MemberWithdrawn", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}

// RLNOwnershipTransferredIterator is returned from FilterOwnershipTransferred and is used to iterate over the raw logs and unpacked data for OwnershipTransferred events raised by the RLN contract.
type RLNOwnershipTransferredIterator struct {
	Event *RLNOwnershipTransferred // Event containing the contract specifics and raw log

	contract *bind.BoundContract // Generic contract to use for unpacking event data
	event    string              // Event name to use for unpacking event data

	logs chan types.Log        // Log channel receiving the found contract events
	sub  ethereum.Subscription // Subscription for errors, completion and termination
	done bool                  // Whether the subscription completed delivering logs
	fail error                 // Occurred error to stop iteration
}

// Next advances the iterator to the subsequent event, returning whether there
// are any more events found. In case of a retrieval or parsing error, false is
// returned and Error() can be queried for the exact failure.
func (it *RLNOwnershipTransferredIterator) Next() bool {
	// If the iterator failed, stop iterating
	if it.fail != nil {
		return false
	}
	// If the iterator completed, deliver directly whatever's available
	if it.done {
		select {
		case log := <-it.logs:
			it.Event = new(RLNOwnershipTransferred)
			if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
				it.fail = err
				return false
			}
			it.Event.Raw = log
			return true

		default:
			return false
		}
	}
	// Iterator still in progress, wait for either a data or an error event
	select {
	case log := <-it.logs:
		it.Event = new(RLNOwnershipTransferred)
		if err := it.contract.UnpackLog(it.Event, it.event, log); err != nil {
			it.fail = err
			return false
		}
		it.Event.Raw = log
		return true

	case err := <-it.sub.Err():
		it.done = true
		it.fail = err
		return it.Next()
	}
}

// Error returns any retrieval or parsing error occurred during filtering.
func (it *RLNOwnershipTransferredIterator) Error() error {
	return it.fail
}

// Close terminates the iteration process, releasing any pending underlying
// resources.
func (it *RLNOwnershipTransferredIterator) Close() error {
	it.sub.Unsubscribe()
	return nil
}

// RLNOwnershipTransferred represents a OwnershipTransferred event raised by the RLN contract.
type RLNOwnershipTransferred struct {
	PreviousOwner common.Address
	NewOwner      common.Address
	Raw           types.Log // Blockchain specific contextual infos
}

// FilterOwnershipTransferred is a free log retrieval operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_RLN *RLNFilterer) FilterOwnershipTransferred(opts *bind.FilterOpts, previousOwner []common.Address, newOwner []common.Address) (*RLNOwnershipTransferredIterator, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _RLN.contract.FilterLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return &RLNOwnershipTransferredIterator{contract: _RLN.contract, event: "OwnershipTransferred", logs: logs, sub: sub}, nil
}

// WatchOwnershipTransferred is a free log subscription operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_RLN *RLNFilterer) WatchOwnershipTransferred(opts *bind.WatchOpts, sink chan<- *RLNOwnershipTransferred, previousOwner []common.Address, newOwner []common.Address) (event.Subscription, error) {

	var previousOwnerRule []interface{}
	for _, previousOwnerItem := range previousOwner {
		previousOwnerRule = append(previousOwnerRule, previousOwnerItem)
	}
	var newOwnerRule []interface{}
	for _, newOwnerItem := range newOwner {
		newOwnerRule = append(newOwnerRule, newOwnerItem)
	}

	logs, sub, err := _RLN.contract.WatchLogs(opts, "OwnershipTransferred", previousOwnerRule, newOwnerRule)
	if err != nil {
		return nil, err
	}
	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case log := <-logs:
				// New log arrived, parse the event and forward to the user
				event := new(RLNOwnershipTransferred)
				if err := _RLN.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
					return err
				}
				event.Raw = log

				select {
				case sink <- event:
				case err := <-sub.Err():
					return err
				case <-quit:
					return nil
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

// ParseOwnershipTransferred is a log parse operation binding the contract event 0x8be0079c531659141344cd1fd0a4f28419497f9722a3daafe3b4186f6b6457e0.
//
// Solidity: event OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
func (_RLN *RLNFilterer) ParseOwnershipTransferred(log types.Log) (*RLNOwnershipTransferred, error) {
	event := new(RLNOwnershipTransferred)
	if err := _RLN.contract.UnpackLog(event, "OwnershipTransferred", log); err != nil {
		return nil, err
	}
	event.Raw = log
	return event, nil
}
