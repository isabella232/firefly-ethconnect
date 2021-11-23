// Copyright 2019 Kaleido

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
	"os"
	"testing"

	ethbinding "github.com/kaleido-io/ethbinding/pkg"
	"github.com/stretchr/testify/assert"
)

func TestPackContractRemovePrefix(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x00",
	}
	compiled, err := packContract("<stdin>:stuff:watsit", contract)
	assert.NoError(err)
	assert.Equal("watsit", compiled.ContractName)
}

func TestPackContractNoPrefix(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x00",
	}
	compiled, err := packContract("thingymobob", contract)
	assert.NoError(err)
	assert.Equal("thingymobob", compiled.ContractName)
}

func TestPackContractFailBadHexCode(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "Not Hex",
	}
	_, err := packContract("", contract)
	assert.Regexp("Decoding bytecode: hex string without 0x prefix", err)
}

func TestPackContractEmpty(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x",
	}
	_, err := packContract("", contract)
	assert.Regexp("Specified contract compiled ok, but did not result in any bytecode: ", err)
}

func TestPackContractFailMarshalABI(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x00",
		Info: ethbinding.ContractInfo{
			AbiDefinition: make(map[bool]bool),
		},
	}
	_, err := packContract("", contract)
	assert.Regexp("Serializing ABI: json: unsupported type: map\\[bool\\]bool", err)
}

func TestPackContractFailUnmarshalABIJSON(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x00",
		Info: ethbinding.ContractInfo{
			AbiDefinition: map[string]string{
				"not": "an ABI",
			},
		},
	}
	_, err := packContract("", contract)
	assert.Regexp("Parsing ABI", err)
}

func TestPackContractFailSerializingDevDoc(t *testing.T) {
	assert := assert.New(t)
	contract := &ethbinding.Contract{
		Code: "0x00",
		Info: ethbinding.ContractInfo{
			DeveloperDoc: make(map[bool]bool),
		},
	}
	_, err := packContract("", contract)
	assert.Regexp("Serializing DevDoc", err.Error())
}

func TestSolcDefaultVersion(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("FLY_SOLC_DEFAULT", "")
	defaultSolc = ""
	solc, err := getSolcExecutable("")
	assert.NoError(err)
	assert.Equal("solc", solc)
	os.Unsetenv("FLY_SOLC_DEFAULT")
}

func TestSolcDefaultVersionEnvVar(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("FLY_SOLC_DEFAULT", "solc123")
	defaultSolc = ""
	solc, err := getSolcExecutable("")
	assert.NoError(err)
	assert.Equal("solc123", solc)
	os.Unsetenv("FLY_SOLC_DEFAULT")
}

func TestSolcCustomVersionValidMajor(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("FLY_SOLC_0_4", "solc04")
	defaultSolc = ""
	solc, err := getSolcExecutable("0.4")
	assert.NoError(err)
	assert.Equal("solc04", solc)
}

func TestSolcCustomVersionValidMinor(t *testing.T) {
	assert := assert.New(t)
	os.Setenv("FLY_SOLC_0_4", "solc04")
	defaultSolc = ""
	solc, err := getSolcExecutable("0.4.23.some interesting things")
	assert.NoError(err)
	assert.Equal("solc04", solc)
}

func TestSolcCustomVersionUnknown(t *testing.T) {
	assert := assert.New(t)
	defaultSolc = ""
	_, err := getSolcExecutable("0.5")
	assert.Regexp("Could not find a configured compiler for requested Solidity major version 0.5", err)
}

func TestSolcCustomVersionInvalid(t *testing.T) {
	assert := assert.New(t)
	defaultSolc = ""
	_, err := getSolcExecutable("0.")
	assert.Regexp("Invalid Solidity version requested for compiler. Ensure the string starts with two dot separated numbers, such as 0.5", err)
}

func TestSolcCompileInvalidVersion(t *testing.T) {
	assert := assert.New(t)
	defaultSolc = ""
	_, err := CompileContract("", "", "zero.four", "")
	assert.Regexp("Invalid Solidity version requested for compiler. Ensure the string starts with two dot separated numbers, such as 0.5", err)
}
