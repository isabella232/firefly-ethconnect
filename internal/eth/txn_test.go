// Copyright 2018, 2021 Kaleido

// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at

//     http://www.apache.org/licenses/LICENSE-2.0

// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package eth

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"reflect"
	"testing"

	"github.com/hyperledger/firefly-ethconnect/internal/ethbind"
	"github.com/hyperledger/firefly-ethconnect/internal/messages"
	ethbinding "github.com/kaleido-io/ethbinding/pkg"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Slim interface for stubbing
type testRPCClient struct {
	mockError       error
	capturedMethod  string
	capturedArgs    []interface{}
	mockError2      error
	capturedMethod2 string
	capturedArgs2   []interface{}
	resultWrangler  func(interface{})
}

func (r *testRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	var retErr error
	if r.capturedMethod == "" {
		r.capturedMethod = method
		r.capturedArgs = args
		retErr = r.mockError
	} else {
		r.capturedMethod2 = method
		r.capturedArgs2 = args
		retErr = r.mockError2
	}
	if r.resultWrangler != nil {
		r.resultWrangler(result)
	}
	return retErr
}

const (
	simpleStorage = "pragma solidity >=0.4.22 <=0.7;\n\ncontract simplestorage {\nuint public storedData;\n\nconstructor(uint initVal) public {\nstoredData = initVal;\n}\n\nfunction set(uint x) public {\nstoredData = x;\n}\n\nfunction get() public view returns (uint retVal) {\nreturn storedData;\n}\n}"
	twoContracts  = "pragma solidity >=0.4.22 <=0.7;\n\ncontract contract1 {function f1() public pure returns (uint retVal) {\nreturn 1;\n}\n}\n\ncontract contract2 {function f2() public pure returns (uint retVal) {\nreturn 2;\n}\n}"
)

func TestNewContractDeployTxnSimpleStorage(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)

	assert.Equal("eth_sendTransaction", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x7b", jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x1c8", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	// The bytecode has the packed parameters appended to the end
	assert.Regexp(".+00000000000000000000000000000000000000000000000000000000000f423f$", jsonSent["data"])

}

func TestNewContractDeployTxnSimpleStorageCalcGas(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.GasPrice = "789"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)

	assert.Equal("eth_estimateGas", rpc.capturedMethod)
	assert.Equal("eth_sendTransaction", rpc.capturedMethod2)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x7b", jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x0", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	// The bytecode has the packed parameters appended to the end
	assert.Regexp(".+00000000000000000000000000000000000000000000000000000000000f423f$", jsonSent["data"])

}

func TestNewContractDeployTxnSimpleStoragePrivate(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "678"
	msg.GasPrice = "0"
	msg.PrivateFrom = "oD76ZRgu6py/WKrsXbtF9++Mf1mxVxzqficE1Uiw6S8="
	msg.PrivateFor = []string{"s6a3mQ8I+rI2ZgHqHZlJaELiJs10HxlZNIwNd669FH4="}
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)

	assert.Equal("eth_estimateGas", rpc.capturedMethod)
	assert.Equal("eth_sendTransaction", rpc.capturedMethod2)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x0", jsonSent["gasPrice"])
	assert.Equal("0x2a6", jsonSent["value"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("oD76ZRgu6py/WKrsXbtF9++Mf1mxVxzqficE1Uiw6S8=", jsonSent["privateFrom"])
	assert.Equal("s6a3mQ8I+rI2ZgHqHZlJaELiJs10HxlZNIwNd669FH4=", jsonSent["privateFor"].([]interface{})[0])

}

func TestNewContractDeployTxnSimpleStoragePrivateOrion(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "678"
	msg.GasPrice = "0"
	msg.PrivateFrom = "oD76ZRgu6py/WKrsXbtF9++Mf1mxVxzqficE1Uiw6S8="
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	tx.PrivacyGroupID = "P8SxRUussJKqZu4+nUkMJpscQeWOR3HqbAXLakatsk8="
	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)

	assert.Equal("eth_estimateGas", rpc.capturedMethod)
	assert.Equal("eea_sendTransaction", rpc.capturedMethod2)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x0", jsonSent["gasPrice"])
	assert.Equal("0x2a6", jsonSent["value"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("oD76ZRgu6py/WKrsXbtF9++Mf1mxVxzqficE1Uiw6S8=", jsonSent["privateFrom"])
	assert.Equal("P8SxRUussJKqZu4+nUkMJpscQeWOR3HqbAXLakatsk8=", jsonSent["privacyGroupId"])

}

func TestNewContractDeployTxnSimpleStoragePrivateOrionMissingPrivateFrom(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "678"
	msg.GasPrice = "0"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	tx.OrionPrivateAPIS = true
	tx.PrivacyGroupID = "s6a3mQ8I+rI2ZgHqHZlJaELiJs10HxlZNIwNd669FH4="
	rpc := testRPCClient{}

	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("private-from is required when submitting private transactions via Orion", err)
}
func TestNewContractDeployTxnSimpleStorageCalcGasFailAndCallSucceeds(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.GasPrice = "789"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	rpc.mockError = fmt.Errorf("pop")
	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("Failed to calculate gas for transaction: pop", err)
}

func TestNewContractDeployTxnSimpleStorageCalcGasFailAndCallFailsAsExpected(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.GasPrice = "789"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	rpc.mockError = fmt.Errorf("estimate gas fails")
	rpc.mockError2 = fmt.Errorf("call fails")
	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("Call failed: call fails", err)
}

func TestNewContractDeployMissingCompiledOrSolidity(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Missing Compiled Code \\+ ABI, or Solidity", err)
}

func TestNewContractDeployPrecompiledSimpleStorage(t *testing.T) {
	assert := assert.New(t)

	c, err := CompileContract(simpleStorage, "simplestorage", "", "")
	assert.NoError(err)

	var msg messages.DeployContract
	msg.Compiled = c.Compiled
	msg.ABI = c.ABI
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)

	assert.Equal("eth_sendTransaction", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x7b", jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x1c8", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	// The bytecode has the packed parameters appended to the end
	assert.Regexp(".+00000000000000000000000000000000000000000000000000000000000f423f$", jsonSent["data"])

}

func TestNewContractDeployTxnBadNonce(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "abc"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Converting supplied 'nonce' to integer", err.Error())
}

func TestNewContractDeployBadValue(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "zzz"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Converting supplied 'value' to big integer", err.Error())
}

func TestNewContractDeployBadGas(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "111"
	msg.Gas = "abc"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Converting supplied 'gas' to integer", err.Error())
}

func TestNewContractDeployBadGasPrice(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{float64(999999)}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "111"
	msg.Gas = "456"
	msg.GasPrice = "abc"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Converting supplied 'gasPrice' to big integer", err.Error())
}

func TestNewContractDeployTxnBadContract(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = "badness"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Solidity compilation failed", err.Error())
}

func TestNewContractDeployStringForNumber(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{"123"}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
}

func TestNewContractDeployTxnBadContractName(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.ContractName = "wrongun"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Contract '<stdin>:wrongun' not found in Solidity source", err.Error())
}
func TestNewContractDeploySpecificContractName(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = twoContracts
	msg.ContractName = "contract1"
	msg.Parameters = []interface{}{}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Nil(err)
}

func TestNewContractDeployMissingNameMultipleContracts(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = twoContracts
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("More than one contract in Solidity file", err.Error())
}

func TestNewContractDeployBadNumber(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{"ABCD"}
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Could not be converted to a number", err.Error())
}

func TestNewContractDeployBadTypeForNumber(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{false}
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Must supply a number or a string", err.Error())
}

func TestNewContractDeployMissingParam(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{}
	_, err := NewContractDeployTxn(&msg, nil)
	assert.Regexp("Requires 1 args \\(supplied=0\\)", err.Error())
}

func testComplexParam(t *testing.T, solidityType string, val interface{}, expectedErr string) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Solidity = "pragma solidity >=0.4.22 <=0.7; contract test {constructor(" + solidityType + " p1) public {}}"
	msg.Parameters = []interface{}{val}
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewContractDeployTxn(&msg, nil)

	if expectedErr == "" {
		assert.Nil(err)
	} else if err == nil {
		assert.Fail("Error expected")
	} else {
		assert.Regexp(expectedErr, err.Error())
	}
}

func TestSolidityUIntParamConversion(t *testing.T) {
	testComplexParam(t, "uint8", float64(123), "")
	testComplexParam(t, "uint8", "123", "")
	testComplexParam(t, "uint16", float64(123), "")
	testComplexParam(t, "uint16", "123", "")
	testComplexParam(t, "uint32", float64(123), "")
	testComplexParam(t, "uint32", "123", "")
	testComplexParam(t, "uint64", float64(123), "")
	testComplexParam(t, "uint64", "123", "")
	testComplexParam(t, "uint64", false, "Must supply a number or a string")
	testComplexParam(t, "uint24", float64(123), "")
	testComplexParam(t, "uint24", "123", "")
	testComplexParam(t, "uint256", float64(123), "")
	testComplexParam(t, "uint256", "123", "")
	testComplexParam(t, "uint256", true, "Must supply a number or a string")
	testComplexParam(t, "uint256", "abc", "Could not be converted to a number")
}
func TestSolidityIntParamConversion(t *testing.T) {
	testComplexParam(t, "int8", float64(123), "")
	testComplexParam(t, "int8", "123", "")
	testComplexParam(t, "int16", float64(123), "")
	testComplexParam(t, "int16", "123", "")
	testComplexParam(t, "int32", float64(123), "")
	testComplexParam(t, "int32", "123", "")
	testComplexParam(t, "int64", float64(123), "")
	testComplexParam(t, "int64", "123", "")
	testComplexParam(t, "int64", false, "Must supply a number or a string")
	testComplexParam(t, "int24", float64(123), "")
	testComplexParam(t, "int24", "123", "")
	testComplexParam(t, "int256", float64(123), "")
	testComplexParam(t, "int256", "123", "")
	testComplexParam(t, "int256", true, "Must supply a number or a string")
	testComplexParam(t, "int256", "abc", "Could not be converted to a number")
}

func TestSolidityIntSliceParamConversion(t *testing.T) {
	testComplexParam(t, "int8[] memory", []float64{123, 456, 789}, "")
	testComplexParam(t, "int8[] memory", []float64{}, "")
	testComplexParam(t, "int256[] memory", []float64{123, 456, 789}, "")
	testComplexParam(t, "int256[] memory", []float64{}, "")
	testComplexParam(t, "int256[] memory", float64(123), "Must supply an array")
	testComplexParam(t, "uint8[] memory", []string{"123"}, "")
	testComplexParam(t, "uint8[] memory", []string{"abc"}, "Could not be converted to a number")
}

func TestSolidityIntArrayParamConversion(t *testing.T) {
	testComplexParam(t, "int8[3] memory", []float64{123, 456, 789}, "")
	testComplexParam(t, "int256[3] memory", []float64{123, 456, 789}, "")
	testComplexParam(t, "int256[3] memory", float64(123), "Must supply an array")
}

func TestSolidityBoolArrayParamConversion(t *testing.T) {
	testComplexParam(t, "bool[] memory", []bool{true, false, true}, "")
	testComplexParam(t, "bool[] memory", []string{"true", "ANYTHING"}, "")
	testComplexParam(t, "bool[] memory", []float64{99}, "Must supply a boolean or a string")
}

func TestSolidityAddressArrayParamConversion(t *testing.T) {
	testComplexParam(t, "address[] memory", []string{"df3394931699709b981a1d6e92f6dd2c93430840", "0x2de6181a8cbfb529207c131d4fc0bba97d3259a9"}, "")
	testComplexParam(t, "address[] memory", []string{"0xfeedbeef"}, "Could not be converted to a hex address")
	testComplexParam(t, "address[] memory", []bool{false}, "Must supply a hex address string")
}

func TestSolidityStringParamConversion(t *testing.T) {
	testComplexParam(t, "string memory", "ok", "")
	testComplexParam(t, "string memory", float64(5), "Must supply a string")
}

func TestSolidityBoolParamConversion(t *testing.T) {
	testComplexParam(t, "bool", true, "")
	testComplexParam(t, "bool", "true", "")
	testComplexParam(t, "bool", float64(5), "Must supply a boolean or a string")
}

func TestSolidityAddressParamConversion(t *testing.T) {
	testComplexParam(t, "address", float64(123), "Must supply a hex address string")
	testComplexParam(t, "address", "123", "Could not be converted to a hex address")
	testComplexParam(t, "address", "0xff", "Could not be converted to a hex address")
	testComplexParam(t, "address", "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", "")
}

func TestSolidityBytesParamConversion(t *testing.T) {
	testComplexParam(t, "bytes32", float64(123), "Must supply a hex string")
	testComplexParam(t, "bytes1", "0f", "")
	testComplexParam(t, "bytes4", "0xfeedbeef", "")
	testComplexParam(t, "bytes memory", []float64{1, 55, 128, 255}, "")
	testComplexParam(t, "bytes memory", []interface{}{float64(128)}, "")
	testComplexParam(t, "bytes memory", []float64{256}, "outside of range for byte")
	testComplexParam(t, "bytes memory", []float64{-1}, "outside of range for byte")
	testComplexParam(t, "bytes memory", []string{"ff"}, "Invalid entry in number array")
	testComplexParam(t, "bytes1", "", "cannot use \\[0\\]uint8 as type \\[1\\]uint8 as argument")
	testComplexParam(t, "bytes16", "0xAA983AD2a0", "cannot use \\[5\\]uint8 as type \\[16\\]uint8 as argument")
	// Below test fails since ethconnect expects bytes32 to be a hex string, should be enhanced to accept plain strings as well
	testComplexParam(t, "bytes32", "john", "cannot use \\[0\\]uint8 as type \\[32\\]uint8 as argument")
	testComplexParam(t, "bytes32", "0x223df1450ad1f2fe995df3df25df18fc7e58b86c87f3b799b8911da1b06d4cef", "")
}

func TestSolidityArrayOfByteArraysParamConversion(t *testing.T) {
	// These types are weird, as they are arrays of arrays of bytes.
	// We do not support HEX strings for these, but the docs explicitly discourage their
	// use in favour of bytes8 etc.
	testComplexParam(t, "byte[8] memory", []string{"fe", "ed", "be", "ef"}, "")
	testComplexParam(t, "byte[] memory", []string{"fe", "ed", "be", "ef"}, "")
	testComplexParam(t, "bytes1[] memory", []string{"fe", "ed", "be", "ef"}, "")
}

func TestTypeNotYetSupported(t *testing.T) {
	assert := assert.New(t)
	var tx Txn
	var m ethbinding.ABIMethod
	functionType, err := ethbind.API.NewType("function", "uint256")
	assert.NoError(err)
	m.Inputs = append(m.Inputs, ethbinding.ABIArgument{Name: "functionType", Type: functionType})
	_, err = tx.generateTypedArgs([]interface{}{"abc"}, &m)
	assert.Regexp("Type '.*' is not yet supported", err)
}

func TestSendTxnABIParam(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{"123", float64(123), "abc", "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", "0xfeedbeef"}
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "uint8",
			},
			{
				Name: "param2",
				Type: "int256",
			},
			{
				Name: "param3",
				Type: "string",
			},
			{
				Name: "param4",
				Type: "address",
			},
			{
				Name: "param5",
				Type: "bytes",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, nil)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("eth_sendTransaction", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x7b", jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x1c8", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	assert.Regexp("0x2898c1bf000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000000007b00000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000aa983ad2a0e0ed8ac639277f37be42f2a5d2618c00000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000000361626300000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000004feedbeef00000000000000000000000000000000000000000000000000000000", jsonSent["data"])
}

func TestSendTxnInlineParam(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = "uint8"
	param1["value"] = "123"

	param2 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param2)
	param2["type"] = "int256"
	param2["value"] = float64(123)

	param3 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param3)
	param3["type"] = "string"
	param3["value"] = "abc"

	param4 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param4)
	param4["type"] = "address"
	param4["value"] = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, nil)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("eth_sendTransaction", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal("0x7b", jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x1c8", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	assert.Regexp("0xe5537abb000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000aa983ad2a0e0ed8ac639277f37be42f2a5d2618c00000000000000000000000000000000000000000000000000000000000000036162630000000000000000000000000000000000000000000000000000000000", jsonSent["data"])
}

func TestSendTxnNilParam(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = "string"
	param1["value"] = nil

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Method 'testFunc' param 0: Cannot supply a null value", err)

}

func TestNewSendTxnMissingParamTypes(t *testing.T) {
	assert := assert.New(t)
	_, err := NewSendTxn(&messages.SendTransaction{
		TransactionCommon: messages.TransactionCommon{
			Parameters: []interface{}{
				map[string]interface{}{
					"wrong": "stuff",
				},
			},
		},
		MethodName: "test",
	}, nil)
	assert.Regexp("Param 0: supplied as an object must have 'type' and 'value' fields", err)
}

func TestCallMethod(t *testing.T) {
	assert := assert.New(t)

	params := []interface{}{}

	param1 := make(map[string]interface{})
	params = append(params, param1)
	param1["type"] = "uint8"
	param1["value"] = "123"

	param2 := make(map[string]interface{})
	params = append(params, param2)
	param2["type"] = "int256"
	param2["value"] = float64(123)

	param3 := make(map[string]interface{})
	params = append(params, param3)
	param3["type"] = "string"
	param3["value"] = "abc"

	param4 := make(map[string]interface{})
	params = append(params, param4)
	param4["type"] = "address"
	param4["value"] = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"

	genMethod := func(params []interface{}) *ethbinding.ABIMethod {
		uint256Type, _ := ethbind.API.ABITypeFor("uint256")
		inputs := make(ethbinding.ABIArguments, len(params))
		for i := range params {
			abiType, _ := ethbind.API.ABITypeFor(params[i].(map[string]interface{})["type"].(string))
			inputs[i] = ethbinding.ABIArgument{
				Type: abiType,
			}
		}
		outputs := ethbinding.ABIArguments{ethbinding.ABIArgument{Name: "retval1", Type: uint256Type}}
		method := ethbind.API.NewMethod("testFunc", "testFunc", ethbinding.Function, "payable", false, true, inputs, outputs)
		return &method
	}

	rpc := &testRPCClient{
		resultWrangler: func(retString interface{}) {
			retVal := "0x000000000000000000000000000000000000000000000000000000000000001"
			reflect.ValueOf(retString).Elem().Set(reflect.ValueOf(retVal))
		},
	}

	res, err := CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "")
	assert.NoError(err)
	assert.Equal(map[string]interface{}{
		"retval1": "1",
	}, res)

	assert.Equal("eth_call", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal(nil, jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal(nil, jsonSent["gas"])
	assert.Equal("0x0", jsonSent["gasPrice"])
	assert.Equal("0x3039", jsonSent["value"])
	assert.Regexp("0xe5537abb000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000aa983ad2a0e0ed8ac639277f37be42f2a5d2618c00000000000000000000000000000000000000000000000000000000000000036162630000000000000000000000000000000000000000000000000000000000", jsonSent["data"])
	assert.Equal("latest", rpc.capturedArgs[1])

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "pending")
	assert.NoError(err)
	assert.Equal("eth_call", rpc.capturedMethod2)
	assert.Equal("pending", rpc.capturedArgs2[1])

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "earliest")
	assert.NoError(err)
	assert.Equal("eth_call", rpc.capturedMethod2)
	assert.Equal("earliest", rpc.capturedArgs2[1])

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "0x1234")
	assert.NoError(err)
	assert.Equal("eth_call", rpc.capturedMethod2)
	assert.Equal("0x1234", rpc.capturedArgs2[1])

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "12345")
	assert.NoError(err)
	assert.Equal("eth_call", rpc.capturedMethod2)
	assert.Equal("0x3039", rpc.capturedArgs2[1])

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), genMethod(params), params, "0")
	assert.NoError(err)
	assert.Equal("eth_call", rpc.capturedMethod2)
	assert.Equal("0x0", rpc.capturedArgs2[1])
}

func TestCallMethodFail(t *testing.T) {
	assert := assert.New(t)

	params := []interface{}{}

	method := &ethbinding.ABIMethod{}
	method.Name = "testFunc"

	rpc := &testRPCClient{
		mockError: fmt.Errorf("pop"),
	}

	_, err := CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), method, params, "")

	assert.Equal("eth_call", rpc.capturedMethod)
	assert.Regexp("Call failed: pop", err)

	_, err = CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), method, params, "ab2345")
	assert.Regexp("Invalid blocknumber. Failed to parse into big integer", err)
}

func TestCallMethodRevert(t *testing.T) {
	assert := assert.New(t)

	params := []interface{}{}

	method := &ethbinding.ABIMethod{}
	method.Name = "testFunc"

	rpc := &testRPCClient{
		resultWrangler: func(retString interface{}) {
			retVal := "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000114d75707065747279206465746563746564000000000000000000000000000000"
			reflect.ValueOf(retString).Elem().Set(reflect.ValueOf(retVal))
		},
	}

	_, err := CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), method, params, "")

	assert.Equal("eth_call", rpc.capturedMethod)
	assert.Regexp("Muppetry detected", err)
}

func TestCallMethodRevertBadStrLen(t *testing.T) {
	assert := assert.New(t)

	params := []interface{}{}

	method := &ethbinding.ABIMethod{}
	method.Name = "testFunc"

	rpc := &testRPCClient{
		resultWrangler: func(retString interface{}) {
			retVal := "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000011111114d75707065747279206465746563746564000000000000000000000000000000"
			reflect.ValueOf(retString).Elem().Set(reflect.ValueOf(retVal))
		},
	}

	_, err := CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), method, params, "")

	assert.Equal("eth_call", rpc.capturedMethod)
	// Should read up to the end of the padding, and not panic
	assert.Regexp("Muppetry detected\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00", err)
}

func TestCallMethodRevertBadBytes(t *testing.T) {
	assert := assert.New(t)

	params := []interface{}{}

	method := &ethbinding.ABIMethod{}
	method.Name = "testFunc"

	rpc := &testRPCClient{
		resultWrangler: func(retString interface{}) {
			retVal := "0x08c379a0000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000002!!!!"
			reflect.ValueOf(retString).Elem().Set(reflect.ValueOf(retVal))
		},
	}

	_, err := CallMethod(context.Background(), rpc, nil,
		"0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		"0x2b8c0ECc76d0759a8F50b2E14A6881367D805832",
		json.Number("12345"), method, params, "")

	assert.Equal("eth_call", rpc.capturedMethod)
	assert.Regexp("EVM reverted. Failed to decode error message", err)
}

func TestCallMethodBadArgs(t *testing.T) {
	assert := assert.New(t)

	rpc := &testRPCClient{
		mockError: fmt.Errorf("pop"),
	}

	_, err := CallMethod(context.Background(), rpc, nil, "badness", "", json.Number(""), &ethbinding.ABIMethod{}, []interface{}{}, "")

	assert.Regexp("Supplied value for 'from' is not a valid hex address", err)
}

func TestSendTxnNodeAssignNonce(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = "uint8"
	param1["value"] = "123"

	param2 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param2)
	param2["type"] = "int256"
	param2["value"] = float64(123)

	param3 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param3)
	param3["type"] = "string"
	param3["value"] = "abc"

	param4 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param4)
	param4["type"] = "address"
	param4["value"] = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, nil)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.NodeAssignNonce = true
	tx.Send(context.Background(), &rpc)
	assert.Equal("eth_sendTransaction", rpc.capturedMethod)
	jsonBytesSent, _ := json.Marshal(rpc.capturedArgs[0])
	var jsonSent map[string]interface{}
	json.Unmarshal(jsonBytesSent, &jsonSent)
	assert.Equal(nil, jsonSent["nonce"])
	assert.Equal("0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c", jsonSent["from"])
	assert.Equal("0x1c8", jsonSent["gas"])
	assert.Equal("0x315", jsonSent["gasPrice"])
	assert.Equal("0x0", jsonSent["value"])
	assert.Regexp("0xe5537abb000000000000000000000000000000000000000000000000000000000000007b000000000000000000000000000000000000000000000000000000000000007b0000000000000000000000000000000000000000000000000000000000000080000000000000000000000000aa983ad2a0e0ed8ac639277f37be42f2a5d2618c00000000000000000000000000000000000000000000000000000000000000036162630000000000000000000000000000000000000000000000000000000000", jsonSent["data"])
}

func TestSendWithTXSignerContractOK(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed: []byte("testbytes"),
		from:   "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
	}

	msg.MethodName = "testFunc"
	msg.From = "hd-u0abcd1234-u0bcde9876-12345"
	msg.Value = "0"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("eth_estimateGas", rpc.capturedMethod)
	assert.Equal("eth_sendRawTransaction", rpc.capturedMethod2)
	assert.Equal("0x746573746279746573", rpc.capturedArgs2[0])
}

func TestSendWithTXSignerOK(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed: []byte("testbytes"),
		from:   "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
	}

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "hd-u0abcd1234-u0bcde9876-12345"
	msg.Value = "0"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("0x2b8c0ECc76d0759a8F50b2E14A6881367D805832", signer.capturedTX.To().String())
	assert.Equal("eth_estimateGas", rpc.capturedMethod)
	assert.Equal("eth_sendRawTransaction", rpc.capturedMethod2)
	assert.Equal("0x746573746279746573", rpc.capturedArgs2[0])
}

func TestSendWithTXSignerFail(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed:  []byte("testbytes"),
		from:    "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		signErr: fmt.Errorf("pop"),
	}

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "hd-u0abcd1234-u0bcde9876-12345"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	tx, err := NewSendTxn(&msg, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("pop", err)
}

func TestSendWithTXSignerFailPrivate(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed:  []byte("testbytes"),
		from:    "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
		signErr: fmt.Errorf("pop"),
	}

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "hd-u0abcd1234-u0bcde9876-12345"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	msg.PrivateFor = []string{"anything"}
	tx, err := NewSendTxn(&msg, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("Signing with mock signer is not currently supported with private transactions", err)
}

func TestNewContractWithTXSignerOK(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed: []byte("testbytes"),
		from:   "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
	}

	msg.From = "hd-u0abcd1234-u0bcde9876-12345"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	msg.Solidity = simpleStorage
	msg.Parameters = []interface{}{"12345"}
	tx, err := NewContractDeployTxn(&msg, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("789", signer.capturedTX.GasPrice().String())
	assert.Equal("eth_sendRawTransaction", rpc.capturedMethod)
	assert.Equal("0x746573746279746573", rpc.capturedArgs[0])
}

func TestNewNilTXSignerOK(t *testing.T) {
	assert := assert.New(t)

	var msg messages.DeployContract
	msg.Parameters = []interface{}{}

	signer := &mockTXSigner{
		signed: []byte("testbytes"),
		from:   "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c",
	}

	// Build a normal SendMessage, but use it to generate a nil transfer
	// transaction - for example to use as a fill transaction attempt.
	// Note the gas and gasPrice are ignored
	tx, err := NewNilTX("hd-u0abcd1234-u0bcde9876-12345", 12345, signer)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{}

	tx.Send(context.Background(), &rpc)
	assert.Equal("0", signer.capturedTX.GasPrice().String())
	assert.Equal(uint64(90000), signer.capturedTX.Gas())
	assert.Equal(uint64(12345), signer.capturedTX.Nonce())
	assert.Equal("eth_sendRawTransaction", rpc.capturedMethod)
}

func TestSendTxnRPFError(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	msg.MethodName = "testFunc"
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	msg.Nonce = "12345"
	tx, err := NewSendTxn(&msg, nil)
	assert.Nil(err)
	msgBytes, _ := json.Marshal(&msg)
	log.Infof(string(msgBytes))

	rpc := testRPCClient{
		mockError: fmt.Errorf("pop"),
	}

	err = tx.Send(context.Background(), &rpc)
	assert.Regexp("pop", err)
}

func TestSendTxnInlineBadParamType(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = "badness"
	param1["value"] = "123"

	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Param 0: Unable to map badness to etherueum type", err.Error())
}

func TestSendTxnInlineMissingParamType(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["value"] = "123"

	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Param 0: supplied as an object must have 'type' and 'value' fields", err.Error())
}

func TestSendTxnInlineMissingParamValue(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = "uint256"

	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Param 0: supplied as an object must have 'type' and 'value' fields", err.Error())
}

func TestSendTxnInlineBadTypeType(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{}

	param1 := make(map[string]interface{})
	msg.Parameters = append(msg.Parameters, param1)
	param1["type"] = false
	param1["value"] = "abcde"

	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Param 0: supplied as an object must be string", err.Error())
}
func TestSendTxnBadInputType(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "badness",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("unsupported arg type: badness", err.Error())
}

func TestSendTxnMissingMethod(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{"123"}
	msg.Method = &ethbinding.ABIElementMarshaling{}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "abc"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Method missing", err.Error())
}
func TestSendTxnBadFrom(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{"123"}
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "uint8",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	msg.To = "0x2b8c0ECc76d0759a8F50b2E14A6881367D805832"
	msg.From = "abc"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Supplied value for 'from' is not a valid hex address", err.Error())
}

func TestSendTxnBadTo(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{"123"}
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "uint8",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	msg.To = "abc"
	msg.From = "0xAA983AD2a0e0eD8ac639277F37be42F2A5d2618c"
	msg.Nonce = "123"
	msg.Value = "0"
	msg.Gas = "456"
	msg.GasPrice = "789"
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("Supplied value for 'to' is not a valid hex address", err.Error())
}

func TestSendTxnBadOutputType(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "uint256",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "badness",
			},
		},
	}
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("unsupported arg type: badness", err.Error())
}

func TestSendBadParams(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{"abc"}
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "int8",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("param 0: Could not be converted to a number", err.Error())
}

func TestSendTxnPackError(t *testing.T) {
	assert := assert.New(t)

	var msg messages.SendTransaction
	msg.Parameters = []interface{}{""}
	msg.Method = &ethbinding.ABIElementMarshaling{
		Name: "testFunc",
		Inputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "param1",
				Type: "bytes1",
			},
		},
		Outputs: []ethbinding.ABIArgumentMarshaling{
			{
				Name: "ret1",
				Type: "uint256",
			},
		},
	}
	_, err := NewSendTxn(&msg, nil)
	assert.Regexp("cannot use \\[0\\]uint8 as type \\[1\\]uint8 as argument", err.Error())
}

func TestProcessRLPBytesValidTypes(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("string")
	t2, _ := ethbind.API.ABITypeFor("int256[]")
	t3, _ := ethbind.API.ABITypeFor("bool")
	t4, _ := ethbind.API.ABITypeFor("bytes1")
	t5, _ := ethbind.API.ABITypeFor("address")
	t6, _ := ethbind.API.ABITypeFor("bytes4")
	t7, _ := ethbind.API.ABITypeFor("uint256")
	t8, _ := ethbind.API.ABITypeFor("int32[]")
	t9, _ := ethbind.API.ABITypeFor("uint32[]")
	methodABI := &ethbinding.ABIMethod{
		Name:   "echoTypes2",
		Inputs: []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{
			{Name: "retval1", Type: t1},
			{Name: "retval2", Type: t2},
			{Name: "retval3", Type: t3},
			{Name: "retval4", Type: t4},
			{Name: "retval5", Type: t5},
			{Name: "retval6", Type: t6},
			{Name: "retval7", Type: t7},
			{Name: "retval8", Type: t8},
			{Name: "retval9", Type: t9},
		},
	}
	rlp, err := methodABI.Outputs.Pack(
		"string 1",
		[]*big.Int{big.NewInt(123)},
		true,
		[1]byte{18},
		[20]byte{18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18, 18},
		[4]byte{18, 18, 18, 18},
		big.NewInt(12345),
		[2]int32{-123, -456},
		[2]uint32{123, 456},
	)
	assert.NoError(err)

	res := ProcessRLPBytes(methodABI.Outputs, rlp)
	assert.Nil(res["error"])

	assert.Equal("string 1", res["retval1"])
	assert.Equal(1, len(res["retval2"].([]interface{})))
	assert.Equal("123", res["retval2"].([]interface{})[0])
	assert.Equal(true, res["retval3"])
	assert.Equal("0x12", res["retval4"])
	assert.Equal("0x1212121212121212121212121212121212121212", res["retval5"])
	assert.Equal("0x12121212", res["retval6"])
	assert.Equal("12345", res["retval7"])
	assert.Equal("-123", res["retval8"].([]interface{})[0])
	assert.Equal("-456", res["retval8"].([]interface{})[1])
	assert.Equal("123", res["retval9"].([]interface{})[0])
	assert.Equal("456", res["retval9"].([]interface{})[1])
}

func TestProcessRLPV2ABIEncodedStructs(t *testing.T) {
	assert := assert.New(t)

	var v2abi ethbinding.ABI
	testABIInput, err := ioutil.ReadFile("../../test/abicoderv2_example.abi.json")
	assert.NoError(err)
	err = json.Unmarshal(testABIInput, &v2abi)
	assert.NoError(err)

	var abiMethod ethbinding.ABIMethod
	for _, m := range v2abi.Methods {
		if m.Name == "inOutType1" {
			abiMethod = m
		}
	}

	input1Map := map[string]interface{}{
		"str1": "test1",
		"val1": "12345",
		"nested": map[string]interface{}{
			"str1":      "test2",
			"str2":      "test3",
			"addr1":     "0x1212121212121212121212121212121212121212",
			"bytearray": "0xfeedbeef",
		},
		"nestarray": []interface{}{
			map[string]interface{}{
				"str1":      "test4",
				"str2":      "test5",
				"addr1":     "0x2121212121212121212121212121212121212121",
				"bytearray": "0x01010101",
			},
		},
	}

	tx := Txn{}
	typedArgs, err := tx.generateTypedArgs([]interface{}{input1Map}, &abiMethod)
	assert.NoError(err)
	t.Logf("typeArgs: %+v", typedArgs)

	rlp, err := abiMethod.Inputs.Pack(typedArgs...)
	assert.NoError(err)
	res := ProcessRLPBytes(abiMethod.Outputs, rlp)
	assert.Nil(res["error"])

	assert.Equal(input1Map, res["out1"])
}

func TestProcessRLPV2ABIEncodedStructsUnasignableVal(t *testing.T) {
	assert := assert.New(t)

	var v2abi ethbinding.ABI
	testABIInput, err := ioutil.ReadFile("../../test/abicoderv2_example.abi.json")
	assert.NoError(err)
	err = json.Unmarshal(testABIInput, &v2abi)
	assert.NoError(err)

	var abiMethod ethbinding.ABIMethod
	for _, m := range v2abi.Methods {
		if m.Name == "inOutType1" {
			abiMethod = m
		}
	}

	input1Map := map[string]interface{}{
		"str1": []interface{}{},
	}

	tx := Txn{}
	_, err = tx.generateTypedArgs([]interface{}{input1Map}, &abiMethod)
	assert.Regexp("Method 'inOutType1' param 0.str1: Must supply a string", err.Error())
}

func TestProcessRLPV2ABIEncodedStructsBadInputType(t *testing.T) {
	assert := assert.New(t)

	var v2abi ethbinding.ABI
	testABIInput, err := ioutil.ReadFile("../../test/abicoderv2_example.abi.json")
	assert.NoError(err)
	err = json.Unmarshal(testABIInput, &v2abi)
	assert.NoError(err)

	var abiMethod ethbinding.ABIMethod
	for _, m := range v2abi.Methods {
		if m.Name == "inOutType1" {
			abiMethod = m
		}
	}

	input1Map := map[string]interface{}{
		"str1":   "ok",
		"val1":   "12345",
		"nested": "Not a map",
	}

	tx := Txn{}
	_, err = tx.generateTypedArgs([]interface{}{input1Map}, &abiMethod)
	assert.Regexp("Method 'inOutType1' param 0.nested is a \\(string,string,address,bytes\\): Must supply an object \\(supplied=string\\)", err)
}

func TestProcessRLPV2ABIEncodedStructsBadNilType(t *testing.T) {
	assert := assert.New(t)

	var v2abi ethbinding.ABI
	testABIInput, err := ioutil.ReadFile("../../test/abicoderv2_example.abi.json")
	assert.NoError(err)
	err = json.Unmarshal(testABIInput, &v2abi)
	assert.NoError(err)

	var abiMethod ethbinding.ABIMethod
	for _, m := range v2abi.Methods {
		if m.Name == "inOutType1" {
			abiMethod = m
		}
	}

	input1Map := map[string]interface{}{}

	tx := Txn{}
	_, err = tx.generateTypedArgs([]interface{}{input1Map}, &abiMethod)
	assert.Regexp("Method inOutType1 param 0: supplied value '<nil>' could not be assigned to 'str1' field \\(string\\)", err)
}

func TestGenerateTupleFromMapBadStructType(t *testing.T) {
	assert := assert.New(t)
	tx := Txn{}
	type random struct{ stuff string }
	tUint, _ := ethbind.API.ABITypeFor("uint256")
	_, err := tx.generateTupleFromMap("method1", "test", &ethbinding.ABIType{
		TupleType:     reflect.TypeOf((*random)(nil)).Elem(), // random type that should never happen
		TupleRawNames: []string{"field1"},
		TupleElems:    []*ethbinding.ABIType{&tUint},
	}, map[string]interface{}{"field1": float64(42)})
	assert.Regexp("Method method1 param test.*supplied value '\\+42' could not be assigned to 'field1' field \\(uint256\\)", err)
}

func TestGenTupleMapOutputBadTypeNonStruct(t *testing.T) {
	assert := assert.New(t)
	type random struct{ stuff string }
	_, err := genTupleMapOutput("test", "random", &ethbinding.ABIType{TupleType: reflect.TypeOf((*string)(nil)).Elem()}, 42)
	assert.Regexp("Unable to process type for test \\(random\\). Expected string. Received 42", err)
}

func TestGenTupleMapOutputBadTypeCountMismatch(t *testing.T) {
	assert := assert.New(t)
	type random struct{}
	_, err := genTupleMapOutput("test", "random", &ethbinding.ABIType{
		TupleType:     reflect.TypeOf((*random)(nil)).Elem(),
		TupleRawNames: []string{"field1", "field2"},
	}, random{})
	assert.Regexp("Unable to process type for test \\(random\\). Expected 2 fields on the structure. Received 0", err)
}

func TestGenTupleMapOutputBadTypeValMismatch(t *testing.T) {
	assert := assert.New(t)
	type random struct{ Field1 string }
	tUint, _ := ethbind.API.ABITypeFor("uint256")
	_, err := genTupleMapOutput("test", "random", &ethbinding.ABIType{
		TupleType:     reflect.TypeOf((*random)(nil)).Elem(),
		TupleRawNames: []string{"field1"},
		TupleElems:    []*ethbinding.ABIType{&tUint},
	}, random{Field1: "stuff"})
	assert.Regexp("Expected number type in JSON/RPC response for test.field1 \\(uint256\\). Received string", err)
}

func TestProcessRLPBytesInvalidNumber(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("int32")
	_, err := mapOutput("test1", "int256", &t1, "not an int")
	assert.Regexp("Expected number type in JSON/RPC response for test1 \\(int256\\). Received string", err)
}

func TestProcessRLPBytesInvalidBool(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("bool")
	_, err := mapOutput("test1", "bool", &t1, "not a bool")
	assert.Regexp("Expected boolean type in JSON/RPC response for test1 \\(bool\\). Received string", err)
}

func TestProcessRLPBytesInvalidString(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("string")
	_, err := mapOutput("test1", "string", &t1, 42)
	assert.Regexp("Expected string array type in JSON/RPC response for test1 \\(string\\). Received int", err)
}

func TestProcessRLPBytesInvalidByteArray(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("address")
	_, err := mapOutput("test1", "address", &t1, 42)
	assert.Regexp("Expected \\[\\]byte type in JSON/RPC response for test1 \\(address\\). Received int", err)
}

func TestProcessRLPBytesInvalidArray(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("int32[]")
	_, err := mapOutput("test1", "int32[]", &t1, 42)
	assert.Regexp("Expected slice type in JSON/RPC response for test1 \\(int32\\[\\]\\). Received int", err)
}

func TestProcessRLPBytesInvalidArrayType(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("int32[]")
	_, err := mapOutput("test1", "int32[]", &t1, []string{"wrong"})
	assert.Regexp("Expected number type in JSON/RPC response for test1\\[0\\] \\(int32\\[\\]\\). Received string", err)
}

func TestProcessRLPBytesInvalidTypeByte(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("bool")
	t1.T = 42
	_, err := mapOutput("test1", "randomness", &t1, 42)
	assert.Regexp("Unable to process type for test1 \\(randomness\\). Received int", err)
}

func TestProcessRLPBytesUnpackFailure(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("string")
	methodABI := &ethbinding.ABIMethod{
		Name:   "echoTypes2",
		Inputs: []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{
			{Name: "retval1", Type: t1},
		},
	}

	res := ProcessRLPBytes(methodABI.Outputs, []byte("this is not the RLP you are looking for"))
	assert.Regexp("Failed to unpack values", res["error"])
}

func TestProcessOutputsTooFew(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("string")
	methodABI := &ethbinding.ABIMethod{
		Name:   "echoTypes2",
		Inputs: []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{
			{Name: "retval1", Type: t1},
		},
	}

	err := processOutputs(methodABI.Outputs, []interface{}{}, make(map[string]interface{}))
	assert.Regexp("Expected 1 in JSON/RPC response. Received 0: \\[\\]", err)
}

func TestProcessOutputsTooMany(t *testing.T) {
	assert := assert.New(t)

	methodABI := &ethbinding.ABIMethod{
		Name:    "echoTypes2",
		Inputs:  []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{},
	}

	err := processOutputs(methodABI.Outputs, []interface{}{"arg1"}, make(map[string]interface{}))
	assert.Regexp("Expected nil in JSON/RPC response. Received: \\[arg1\\]", err)
}

func TestProcessOutputsDefaultName(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("string")
	methodABI := &ethbinding.ABIMethod{
		Name:   "anonReturn",
		Inputs: []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{
			{Name: "", Type: t1},
			{Name: "", Type: t1},
		},
	}

	retval := make(map[string]interface{})
	err := processOutputs(methodABI.Outputs, []interface{}{"arg1", "arg2"}, retval)
	assert.NoError(err)
	assert.Equal("arg1", retval["output"])
	assert.Equal("arg2", retval["output1"])
}
func TestProcessOutputsBadArgs(t *testing.T) {
	assert := assert.New(t)

	t1, _ := ethbind.API.ABITypeFor("int32[]")
	methodABI := &ethbinding.ABIMethod{
		Name:   "echoTypes2",
		Inputs: []ethbinding.ABIArgument{},
		Outputs: []ethbinding.ABIArgument{
			{Name: "retval1", Type: t1},
		},
	}

	err := processOutputs(methodABI.Outputs, []interface{}{"arg1"}, make(map[string]interface{}))
	assert.Regexp("Expected slice type in JSON/RPC response for retval1 \\(int32\\[\\]\\). Received string", err)
}

func TestGetTransactionInfoFail(t *testing.T) {
	assert := assert.New(t)
	rpc := testRPCClient{}

	info, err := GetTransactionInfo(context.Background(), &rpc, "0x12345")
	assert.Regexp("Failed to query transaction: 0x12345", err)
	assert.Nil(info)
}

func TestGetTransactionInfoError(t *testing.T) {
	assert := assert.New(t)
	rpc := testRPCClient{
		mockError: fmt.Errorf("pop"),
	}

	info, err := GetTransactionInfo(context.Background(), &rpc, "0x12345")
	assert.Regexp("pop", err)
	assert.Nil(info)
}

func TestGetTransactionInfo(t *testing.T) {
	assert := assert.New(t)
	rpc := testRPCClient{
		resultWrangler: func(txn interface{}) {
			json.Unmarshal([]byte(`{"input":"0x01"}`), &txn)
		},
	}

	info, err := GetTransactionInfo(context.Background(), &rpc, "0x12345")
	assert.NoError(err)
	assert.NotNil(info)
	assert.Equal(ethbinding.HexBytes{1}, *info.Input)
}

func TestDecodeInputsBadSignature(t *testing.T) {
	assert := assert.New(t)
	method := ethbinding.ABIMethod{
		ID:     []byte{1, 2, 3, 4},
		Inputs: ethbinding.ABIArguments{},
	}
	inputs := ethbinding.HexBytes{1}

	args, err := DecodeInputs(&method, &inputs)
	assert.Regexp("Method signature did not match", err)
	assert.Nil(args)
}

func TestDecodeInputsNoMatch(t *testing.T) {
	assert := assert.New(t)
	method := ethbinding.ABIMethod{
		ID:     []byte{1, 2, 3, 4},
		Inputs: ethbinding.ABIArguments{},
	}
	inputs := ethbinding.HexBytes{1, 2, 3, 5}

	args, err := DecodeInputs(&method, &inputs)
	assert.Regexp("Method signature did not match", err)
	assert.Nil(args)
}

func TestDecodeInputs(t *testing.T) {
	assert := assert.New(t)
	method := ethbinding.ABIMethod{
		ID: []byte{1, 2, 3, 4},
		Inputs: ethbinding.ABIArguments{
			{
				Name: "arg1",
				Type: ethbinding.ABIType{},
			},
		},
	}
	inputs := ethbinding.HexBytes{1, 2, 3, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1}
	expectedArgs := make(map[string]interface{}, 0)
	expectedArgs["arg1"] = "1"

	args, err := DecodeInputs(&method, &inputs)
	assert.NoError(err)
	assert.Equal(expectedArgs, args)
}
