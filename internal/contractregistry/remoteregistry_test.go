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
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"testing"

	"github.com/hyperledger/firefly-ethconnect/internal/ethbind"
	"github.com/hyperledger/firefly-ethconnect/internal/kvstore"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func tempdir() string {
	dir, _ := ioutil.TempDir("", "fly")
	return dir
}

func cleanup(dir string) {
	os.RemoveAll(dir)
}

func TestNewRemoteRegistryDefaultPropNames(t *testing.T) {
	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix:  "http://www.example1.com/",
		InstanceURLPrefix: "http://www.example2.com/",
	})
	rr := r.(*remoteRegistry)
	assert.Equal("http://www.example1.com/", rr.conf.GatewayURLPrefix)
	assert.Equal("http://www.example2.com/", rr.conf.InstanceURLPrefix)
	assert.Equal(defaultIDProp, rr.conf.PropNames.ID)
	assert.Equal(defaultNameProp, rr.conf.PropNames.Name)
	assert.Equal(defaultABIProp, rr.conf.PropNames.ABI)
	assert.Equal(defaultBytecodeProp, rr.conf.PropNames.Bytecode)
	assert.Equal(defaultDevdocProp, rr.conf.PropNames.Devdoc)
	assert.Equal(defaultDeployableProp, rr.conf.PropNames.Deployable)
	assert.Equal(defaultAddressProp, rr.conf.PropNames.Address)
}

func TestNewRemoteRegistryCustomPropNames(t *testing.T) {
	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix:  "http://www.example1.com",
		InstanceURLPrefix: "http://www.example2.com",
		PropNames: RemoteRegistryPropNamesConf{
			ID:         "idProp",
			Name:       "nameProp",
			ABI:        "abiProp",
			Bytecode:   "bytecodeProp",
			Devdoc:     "devdocsProp",
			Deployable: "deployableProp",
			Address:    "addressProp",
		},
	})
	rr := r.(*remoteRegistry)
	assert.Equal("http://www.example1.com/", rr.conf.GatewayURLPrefix)
	assert.Equal("http://www.example2.com/", rr.conf.InstanceURLPrefix)
	assert.Equal("idProp", rr.conf.PropNames.ID)
	assert.Equal("nameProp", rr.conf.PropNames.Name)
	assert.Equal("abiProp", rr.conf.PropNames.ABI)
	assert.Equal("bytecodeProp", rr.conf.PropNames.Bytecode)
	assert.Equal("devdocsProp", rr.conf.PropNames.Devdoc)
	assert.Equal("deployableProp", rr.conf.PropNames.Deployable)
	assert.Equal("addressProp", rr.conf.PropNames.Address)
}

func TestRemoteRegistryInitDB(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)

	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{
		CacheDB: path.Join(dir, "test"),
	})
	rr := r.(*remoteRegistry)

	err := rr.Init()
	assert.NoError(err)
	rr.Close()
}

func TestRemoteRegistryInitBadDB(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)

	assert := assert.New(t)

	db := path.Join(dir, "test")
	ioutil.WriteFile(db, []byte{}, 0644)
	r := NewRemoteRegistry(&RemoteRegistryConf{
		CacheDB: db,
	})
	rr := r.(*remoteRegistry)

	err := rr.Init()
	assert.Regexp("Failed to initialize cache for remote registry", err.Error())
	rr.Close()
}

func TestRemoteRegistryloadFactoryForGatewaySuccess(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		testDataBytes, _ := ioutil.ReadFile("../../test/simpleevents.solc.output.json")
		res.WriteHeader(200)
		res.Write(testDataBytes)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	res, err := rr.LoadFactoryForGateway("testid", false)
	assert.NoError(err)
	assert.NotEmpty(res.Compiled)
	runtimeABI, err := ethbind.API.ABIMarshalingToABIRuntime(res.ABI)
	assert.NoError(err)
	assert.Equal("set", runtimeABI.Methods["set"].Name)
	assert.Contains(res.DevDoc, "set")
}

func TestRemoteRegistryloadFactoryForGatewayCached(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)

	assert := assert.New(t)

	callCount := 0
	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		callCount++
		assert.Equal("testid", parms.ByName("id"))
		testDataBytes, _ := ioutil.ReadFile("../../test/simpleevents.solc.output.json")
		res.WriteHeader(200)
		res.Write(testDataBytes)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		CacheDB:          path.Join(dir, "testdb"),
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)
	rr.Init()
	defer rr.Close()

	res1, err := rr.LoadFactoryForGateway("testid", false)
	assert.NoError(err)
	res2, err := rr.LoadFactoryForGateway("testid", false)
	assert.NoError(err)
	assert.Equal(1, callCount)
	assert.Equal(res1.Headers.ID, res2.Headers.ID)
	assert.Equal(res1.ABI, res2.ABI)
	assert.Equal(res1.DevDoc, res2.DevDoc)
	assert.Equal(res1.Compiled, res2.Compiled)

	// Force reload
	res3, err := rr.LoadFactoryForGateway("testid", true)
	assert.NoError(err)
	assert.Equal(res1.Headers.ID, res3.Headers.ID)
	assert.Equal(2, callCount)
}

func TestRemoteRegistryRegisterInstanceSuccess(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.POST("/somepath", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		var bodyMap map[string]string
		json.NewDecoder(req.Body).Decode(&bodyMap)
		assert.Equal("testid", bodyMap["name"])
		assert.Equal("12345", bodyMap["address"])
		assert.Equal("application/json", req.Header.Get("content-type"))
		res.WriteHeader(204)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		InstanceURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	err := rr.RegisterInstance("testid", "12345")
	assert.NoError(err)
}

func TestRemoteRegistryRegisterInstanceFail(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.POST("/somepath", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		res.WriteHeader(500)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		InstanceURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	err := rr.RegisterInstance("testid", "12345")
	assert.Regexp("Failed to register instance in remote registry", err)
}

func TestRemoteRegistryRegisterNoInstanceURL(t *testing.T) {
	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	err := rr.RegisterInstance("testid", "12345")
	assert.Regexp("No remote registry is configured", err)
}

func TestRemoteRegistryLoadFactoryMissingID(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{

    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("'id' missing in Contract registry response", err)
}

func TestRemoteRegistryLoadFactoryMissingABI(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345"
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("'abi' missing in Contract registry response", err)
}

func TestRemoteRegistryLoadFactoryBadABIJSON(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345",
      "abi": "!JSON"
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("Error processing contract registry response", err)
}

func TestRemoteRegistryLoadFactoryMissingDevDoc(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345",
      "abi": "[]"
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("'devdoc' missing in Contract registry response", err)
}

func TestRemoteRegistryLoadFactoryBadDevDoc(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345",
      "abi": "[]",
      "devdoc": 123
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("'devdoc' not a string in Contract registry response", err)
}

func TestRemoteRegistryLoadFactoryEmptyBytecode(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345",
      "abi": "[]",
      "devdoc": null,
      "bin": ""
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("'bin' empty \\(or null\\) in Contract registry response", err)
}

func TestRemoteRegistryLoadFactoryBadBytecode(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`{
      "id": "12345",
      "abi": "[]",
      "devdoc": "",
      "bin": "!HEX"
    }`))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("Error processing contract registry response", err)
}

func TestRemoteRegistryLoadFactoryErrorStatusGeneric(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(500)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("Could not process Contract registry \\[500\\] response", err)
}

func TestRemoteRegistryLoadFactoryErrorStatus(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(500)
		res.Write([]byte("{\"errorMessage\":\"poof\"}"))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("Contract registry returned \\[500\\]: poof", err)
}

func TestRemoteRegistryLoadFactoryNotFound(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(404)
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	res, err := rr.LoadFactoryForGateway("testid", false)
	assert.NoError(err)
	assert.Nil(res)
}

func TestRemoteRegistryLoadFactoryBadBody(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte("!JSON"))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		GatewayURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	_, err := rr.LoadFactoryForGateway("testid", false)
	assert.Regexp("Could not process Contract registry \\[200\\] response", err)
}

func TestRemoteRegistryLoadFactoryNOOP(t *testing.T) {
	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{})
	rr := r.(*remoteRegistry)

	res, err := rr.LoadFactoryForGateway("testid", false)
	assert.NoError(err)
	assert.Nil(res)
}

func TestRemoteRegistryloadFactoryForInstanceSuccess(t *testing.T) {
	assert := assert.New(t)

	router := &httprouter.Router{}
	router.GET("/somepath/:id", func(res http.ResponseWriter, req *http.Request, parms httprouter.Params) {
		assert.Equal("testid", parms.ByName("id"))
		res.WriteHeader(200)
		res.Write([]byte(`
      {
        "address": "0x35344E187D669D930C9d513AaC63Ae204fC03C18",
        "id": "12345",
        "abi": "[]",
        "devdoc": "",
        "bin": "0x"
      }
    `))
	})
	server := httptest.NewServer(router)
	defer server.Close()

	r := NewRemoteRegistry(&RemoteRegistryConf{
		InstanceURLPrefix: server.URL + "/somepath",
		PropNames: RemoteRegistryPropNamesConf{
			Bytecode: "bin",
		},
	})
	rr := r.(*remoteRegistry)

	res, err := rr.LoadFactoryForInstance("testid", false)
	assert.NoError(err)
	assert.Equal(res.Address, "35344e187d669d930c9d513aac63ae204fc03c18")
}

func TestRemoteRegistryLoadInstanceNOOP(t *testing.T) {
	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{})
	rr := r.(*remoteRegistry)

	res, err := rr.LoadFactoryForInstance("testid", false)
	assert.NoError(err)
	assert.Nil(res)
}

func TestRemoteRegistryLoadFactoryFromCacheDBBadBytes(t *testing.T) {
	dir := tempdir()
	defer cleanup(dir)

	assert := assert.New(t)

	r := NewRemoteRegistry(&RemoteRegistryConf{
		CacheDB: path.Join(dir, "testdb"),
	})
	rr := r.(*remoteRegistry)
	rr.Init()
	defer rr.Close()

	rr.db.Put("testid", []byte("!Bad JSON!"))

	msg := rr.loadFactoryFromCacheDB("testid")
	assert.Nil(msg)
}

func TestRemoteRegistryStoreFactoryToCacheDBBadObj(t *testing.T) {
	r := NewRemoteRegistry(&RemoteRegistryConf{})
	rr := r.(*remoteRegistry)
	mockKV := kvstore.NewMockKV(nil)
	rr.db = mockKV
	mockKV.StoreErr = fmt.Errorf("pop")
	rr.storeFactoryToCacheDB("testid", nil)
}
