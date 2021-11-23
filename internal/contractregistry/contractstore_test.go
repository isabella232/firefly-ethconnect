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

package contractregistry

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path"
	"testing"
	"time"

	"github.com/go-openapi/spec"
	"github.com/hyperledger/firefly-ethconnect/internal/eth"
	"github.com/hyperledger/firefly-ethconnect/internal/messages"
	"github.com/stretchr/testify/assert"
)

type mockRR struct {
	deployMsg *DeployContractWithAddress
	err       error
}

func (rr *mockRR) LoadFactoryForGateway(id string, refresh bool) (*messages.DeployContract, error) {
	if rr.deployMsg == nil {
		return nil, rr.err
	}
	return rr.deployMsg.Contract, rr.err
}
func (rr *mockRR) LoadFactoryForInstance(id string, refresh bool) (*DeployContractWithAddress, error) {
	return rr.deployMsg, rr.err
}
func (rr *mockRR) RegisterInstance(lookupStr, address string) error {
	return rr.err
}
func (rr *mockRR) Close()      {}
func (rr *mockRR) Init() error { return nil }

var simpleEventsSol string

func simpleEventsSource() string {
	if simpleEventsSol == "" {
		simpleEventsBytes, _ := ioutil.ReadFile("../../test/simpleevents.sol")
		simpleEventsSol = string(simpleEventsBytes)
	}
	return simpleEventsSol
}

func newTestDeployMsg(t *testing.T, addr string) *DeployContractWithAddress {
	compiled, err := eth.CompileContract(simpleEventsSource(), "SimpleEvents", "", "")
	assert.NoError(t, err)
	return &DeployContractWithAddress{
		Contract: &messages.DeployContract{ABI: compiled.ABI},
		Address:  addr,
	}
}

func TestLoadDeployMsgOK(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)
	deployFile := path.Join(dir, "abi_abi1.deploy.json")

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	goodMsg := &messages.DeployContract{}
	deployBytes, _ := json.Marshal(goodMsg)
	cs.(*contractStore).abiIndex["abi1"] = &ABIInfo{}
	ioutil.WriteFile(deployFile, deployBytes, 0644)
	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.NoError(err)

	// verify cache hit
	assert.Equal(1, cs.(*contractStore).abiCache.Len())
	ioutil.WriteFile(deployFile, []byte{}, 0644)
	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.NoError(err)
}

func TestLoadDeployMsgMissing(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.Regexp("No ABI found with ID abi1", err.Error())
}

func TestLoadDeployMsgFileMissing(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	cs.(*contractStore).abiIndex["abi1"] = &ABIInfo{}
	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.Regexp("Failed to load ABI with ID abi1", err.Error())
}

func TestLoadDeployMsgFailure(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	cs.(*contractStore).abiIndex["abi1"] = &ABIInfo{}
	ioutil.WriteFile(path.Join(dir, "abi_abi1.deploy.json"), []byte(":bad json"), 0644)
	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.Regexp("Failed to parse ABI with ID abi1", err.Error())
}

func TestLoadDeployMsgRemoteLookupNotFound(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "abi1",
	}, false)
	assert.Regexp("No ABI found with ID abi1", err)
}

func TestStoreABIWriteFail(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: path.Join(dir, "badpath")}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	i := &ContractInfo{
		Address: "req1",
	}
	err = cs.(*contractStore).storeContractInfo(i)
	assert.Regexp("Failed to write ABI JSON", err.Error())
}

func TestLoadABIForInstanceUnknown(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: path.Join(dir, "badpath")}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	_, err = cs.GetContractByAddress("invalid")
	assert.Regexp("No contract instance registered with address invalid", err.Error())
}

func TestLoadABIBadData(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	ioutil.WriteFile(path.Join(dir, "badness.abi.json"), []byte(":not json"), 0644)
	_, err = cs.GetABI(ABILocation{
		ABIType: LocalABI,
		Name:    "badness",
	}, false)
	assert.Regexp("No ABI found with ID badness", err.Error())
}

func TestAddFileToContractIndexBadFileSwallowsError(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)
	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, nil)
	cs.(*contractStore).addFileToContractIndex("", "badness")
}

func TestAddFileToContractIndexBadDataSwallowsError(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)
	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, nil)
	fileName := path.Join(dir, "badness")
	ioutil.WriteFile(fileName, []byte("!JSON"), 0644)
	cs.(*contractStore).addFileToContractIndex("", fileName)
}

func TestAddFileToABIIndexBadFileSwallowsError(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)
	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, nil)
	cs.(*contractStore).addFileToABIIndex("", "badness", time.Now().UTC())
}

func TestCheckNameAvailableRRDuplicate(t *testing.T) {
	assert := assert.New(t)

	mrr := &mockRR{
		deployMsg: newTestDeployMsg(t, "12345"),
	}
	cs := NewContractStore(&ContractStoreConf{BaseURL: "http://localhost/api/v1"}, mrr)

	err := cs.CheckNameAvailable("lobster", true)
	assert.Regexp("Contract address 12345 is already registered for name 'lobster'", err)
}

func TestCheckNameAvailableRRFail(t *testing.T) {
	assert := assert.New(t)

	mrr := &mockRR{
		err: fmt.Errorf("pop"),
	}
	cs := NewContractStore(&ContractStoreConf{BaseURL: "http://localhost/api/v1"}, mrr)

	err := cs.CheckNameAvailable("lobster", true)
	assert.Regexp("pop", err)
}

func TestBuildIndex(t *testing.T) {
	assert := assert.New(t)
	dir := tempdir()
	defer cleanup(dir)

	// Migration of legacy contract interfaces

	var emptySwagger spec.Swagger
	swaggerBytes, _ := json.Marshal(&emptySwagger)
	ioutil.WriteFile(path.Join(dir, "contract_0123456789abcdef0123456789abcdef01234567.swagger.json"), swaggerBytes, 0644)

	okSwagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title: "good one",
				},
			},
		},
	}
	okSwagger.Info.AddExtension("x-firefly-deployment-id", "840b629f-2e46-413b-9671-553a886ca7bb")
	swaggerBytes, _ = json.Marshal(&okSwagger)
	ioutil.WriteFile(path.Join(dir, "contract_123456789abcdef0123456789abcdef012345678.swagger.json"), swaggerBytes, 0644)

	regSwagger := spec.Swagger{
		SwaggerProps: spec.SwaggerProps{
			Info: &spec.Info{
				InfoProps: spec.InfoProps{
					Title: "good one",
				},
			},
		},
	}
	regSwagger.Info.AddExtension("x-firefly-deployment-id", "840b629f-2e46-413b-9671-553a886ca7bb")
	regSwagger.Info.AddExtension("x-firefly-registered-name", "migratedcontract")
	swaggerBytes, _ = json.Marshal(&regSwagger)
	ioutil.WriteFile(path.Join(dir, "contract_23456789abcdef0123456789abcdef0123456789.swagger.json"), swaggerBytes, 0644)

	ioutil.WriteFile(path.Join(dir, "contract_3456789abcdef0123456789abcdef01234567890.swagger.json"), []byte(":bad swagger"), 0644)

	// New contract interfaces
	info1 := &ContractInfo{
		Address:      "456789abcdef0123456789abcdef012345678901",
		ABI:          "840b629f-2e46-413b-9671-553a886ca7bb",
		Path:         "/contracts/456789abcdef0123456789abcdef012345678901",
		SwaggerURL:   "http://localhost:8080/contracts/456789abcdef0123456789abcdef012345678901?swagger",
		RegisteredAs: "",
		TimeSorted: messages.TimeSorted{
			CreatedISO8601: time.Now().UTC().Format(time.RFC3339),
		},
	}
	info1Bytes, _ := json.Marshal(info1)
	ioutil.WriteFile(path.Join(dir, "contract_456789abcdef0123456789abcdef012345678901.instance.json"), info1Bytes, 0644)
	info2 := &ContractInfo{
		Address:      "56789abcdef0123456789abcdef0123456789012",
		ABI:          "840b629f-2e46-413b-9671-553a886ca7bb",
		Path:         "/contracts/somecontract",
		SwaggerURL:   "http://localhost:8080/contracts/somecontract?swagger",
		RegisteredAs: "somecontract",
		TimeSorted: messages.TimeSorted{
			CreatedISO8601: time.Now().UTC().Format(time.RFC3339),
		},
	}
	info2Bytes, _ := json.Marshal(info2)
	ioutil.WriteFile(path.Join(dir, "contract_56789abcdef0123456789abcdef0123456789012.instance.json"), info2Bytes, 0644)

	deployMsg := &messages.DeployContract{
		ContractName: "abideployable",
	}
	deployBytes, _ := json.Marshal(&deployMsg)
	ioutil.WriteFile(path.Join(dir, "abi_840b629f-2e46-413b-9671-553a886ca7bb.deploy.json"), deployBytes, 0644)
	ioutil.WriteFile(path.Join(dir, "abi_e27be4cf-6ae2-411e-8088-db2992618938.deploy.json"), deployBytes, 0644)
	ioutil.WriteFile(path.Join(dir, "abi_519526b2-0879-41f4-93c0-09acaa62e2da.deploy.json"), []byte(":bad json"), 0644)

	cs := NewContractStore(&ContractStoreConf{StoragePath: dir}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	contracts := cs.ListContracts()
	assert.Equal(4, len(contracts))
	assert.Equal("123456789abcdef0123456789abcdef012345678", contracts[0].(*ContractInfo).Address)
	assert.Equal("23456789abcdef0123456789abcdef0123456789", contracts[1].(*ContractInfo).Address)
	assert.Equal("456789abcdef0123456789abcdef012345678901", contracts[2].(*ContractInfo).Address)
	assert.Equal("56789abcdef0123456789abcdef0123456789012", contracts[3].(*ContractInfo).Address)

	info, err := cs.GetContractByAddress("123456789abcdef0123456789abcdef012345678")
	assert.NoError(err)
	assert.Equal("123456789abcdef0123456789abcdef012345678", info.Address)

	somecontractAddr, err := cs.ResolveContractAddress("somecontract")
	assert.NoError(err)
	assert.Equal("56789abcdef0123456789abcdef0123456789012", somecontractAddr)

	migratedcontractAddr, err := cs.ResolveContractAddress("migratedcontract")
	assert.NoError(err)
	assert.Equal("23456789abcdef0123456789abcdef0123456789", migratedcontractAddr)

	abis := cs.ListABIs()
	assert.Equal(2, len(abis))
	assert.Equal("840b629f-2e46-413b-9671-553a886ca7bb", abis[0].(*ABIInfo).ID)
	assert.Equal("e27be4cf-6ae2-411e-8088-db2992618938", abis[1].(*ABIInfo).ID)
}

func TestGetABIRemoteGateway(t *testing.T) {
	assert := assert.New(t)

	mrr := &mockRR{
		deployMsg: &DeployContractWithAddress{
			Contract: &messages.DeployContract{
				Description: "description",
			},
			Address: "address",
		},
	}

	cs := NewContractStore(&ContractStoreConf{}, mrr)
	err := cs.Init()
	assert.NoError(err)

	location := ABILocation{ABIType: RemoteGateway}
	deployMsg, err := cs.GetABI(location, false)
	assert.NoError(err)
	assert.Equal("", deployMsg.Address)
	assert.Equal("description", deployMsg.Contract.Description)
}

func TestGetABIRemoteInstance(t *testing.T) {
	assert := assert.New(t)

	mrr := &mockRR{
		deployMsg: &DeployContractWithAddress{
			Contract: &messages.DeployContract{
				Description: "description",
			},
			Address: "address",
		},
	}

	cs := NewContractStore(&ContractStoreConf{}, mrr)
	err := cs.Init()
	assert.NoError(err)

	location := ABILocation{ABIType: RemoteInstance}
	deployMsg, err := cs.GetABI(location, false)
	assert.NoError(err)
	assert.Equal("address", deployMsg.Address)
	assert.Equal("description", deployMsg.Contract.Description)

	// verify cache hit
	assert.Equal(1, cs.(*contractStore).abiCache.Len())
	mrr.deployMsg = nil
	deployMsg, err = cs.GetABI(location, false)
	assert.NoError(err)
	assert.Equal("address", deployMsg.Address)
	assert.Equal("description", deployMsg.Contract.Description)
}

func TestGetABIRemoteInstanceFail(t *testing.T) {
	assert := assert.New(t)

	cs := NewContractStore(&ContractStoreConf{}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	location := ABILocation{ABIType: RemoteInstance}
	deployMsg, err := cs.GetABI(location, false)
	assert.NoError(err)
	assert.Nil(deployMsg)
}

func TestGetABILocalFail(t *testing.T) {
	assert := assert.New(t)

	cs := NewContractStore(&ContractStoreConf{}, &mockRR{})
	err := cs.Init()
	assert.NoError(err)

	location := ABILocation{ABIType: LocalABI, Name: "test"}
	deployMsg, err := cs.GetABI(location, false)
	assert.Regexp("No ABI found with ID test", err)
	assert.Nil(deployMsg)
}

func TestIsRemote(t *testing.T) {
	assert := assert.New(t)

	result := IsRemote(messages.CommonHeaders{
		Context: map[string]interface{}{
			RemoteRegistryContextKey: true,
		},
	})
	assert.Equal(true, result)

	result = IsRemote(messages.CommonHeaders{
		Context: map[string]interface{}{
			RemoteRegistryContextKey: false,
		},
	})
	assert.Equal(false, result)

	result = IsRemote(messages.CommonHeaders{
		Context: map[string]interface{}{},
	})
	assert.Equal(false, result)
}
