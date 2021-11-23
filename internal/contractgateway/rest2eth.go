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

package contractgateway

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/hyperledger/firefly-ethconnect/internal/auth"
	"github.com/hyperledger/firefly-ethconnect/internal/contractregistry"
	"github.com/hyperledger/firefly-ethconnect/internal/errors"
	ethconnecterrors "github.com/hyperledger/firefly-ethconnect/internal/errors"
	"github.com/hyperledger/firefly-ethconnect/internal/eth"
	"github.com/hyperledger/firefly-ethconnect/internal/ethbind"
	"github.com/hyperledger/firefly-ethconnect/internal/events"
	"github.com/hyperledger/firefly-ethconnect/internal/messages"
	"github.com/hyperledger/firefly-ethconnect/internal/tx"
	"github.com/hyperledger/firefly-ethconnect/internal/utils"
	"github.com/julienschmidt/httprouter"
	ethbinding "github.com/kaleido-io/ethbinding/pkg"

	log "github.com/sirupsen/logrus"
)

// REST2EthAsyncDispatcher is passed in to process messages over a streaming system with
// a receipt store. Only used for POST methods, when fly-sync is not set to true
type REST2EthAsyncDispatcher interface {
	DispatchMsgAsync(ctx context.Context, msg map[string]interface{}, ack, immediateReceipt bool) (*messages.AsyncSentMsg, error)
}

// rest2EthSyncDispatcher abstracts the processing of the transactions and queries
// synchronously. We perform those within this package.
type rest2EthSyncDispatcher interface {
	DispatchSendTransactionSync(ctx context.Context, msg *messages.SendTransaction, replyProcessor rest2EthReplyProcessor)
	DispatchDeployContractSync(ctx context.Context, msg *messages.DeployContract, replyProcessor rest2EthReplyProcessor)
}

// rest2EthReplyProcessor interface
type rest2EthReplyProcessor interface {
	ReplyWithError(err error)
	ReplyWithReceipt(receipt messages.ReplyWithHeaders)
	ReplyWithReceiptAndError(receipt messages.ReplyWithHeaders, err error)
}

// rest2eth provides the HTTP <-> messages translation and dispatches for processing
type rest2eth struct {
	gw              SmartContractGateway
	cr              contractregistry.ContractResolver
	rpc             eth.RPCClient
	processor       tx.TxnProcessor
	asyncDispatcher REST2EthAsyncDispatcher
	syncDispatcher  rest2EthSyncDispatcher
	subMgr          events.SubscriptionManager
}

type restAsyncMsg struct {
	OK string `json:"ok"`
}

type restReceiptAndError struct {
	Message string `json:"error"`
	messages.ReplyWithHeaders
}

// rest2EthInflight is instantiated for each async reply in flight
type rest2EthSyncResponder struct {
	r      *rest2eth
	res    http.ResponseWriter
	req    *http.Request
	done   bool
	waiter *sync.Cond
}

var addrCheck = regexp.MustCompile("^(0x)?[0-9a-z]{40}$")

func (i *rest2EthSyncResponder) ReplyWithError(err error) {
	i.r.restErrReply(i.res, i.req, err, 500)
	i.done = true
	i.waiter.Broadcast()
	return
}

func (i *rest2EthSyncResponder) ReplyWithReceiptAndError(receipt messages.ReplyWithHeaders, err error) {
	status := 500
	reply, _ := json.MarshalIndent(&restReceiptAndError{err.Error(), receipt}, "", "  ")
	log.Infof("<-- %s %s [%d]", i.req.Method, i.req.URL, status)
	log.Debugf("<-- %s", reply)
	i.res.Header().Set("Content-Type", "application/json")
	i.res.WriteHeader(status)
	i.res.Write(reply)
	i.done = true
	i.waiter.Broadcast()
	return
}

func (i *rest2EthSyncResponder) ReplyWithReceipt(receipt messages.ReplyWithHeaders) {
	txReceiptMsg := receipt.IsReceipt()
	if txReceiptMsg != nil && txReceiptMsg.ContractAddress != nil {
		if err := i.r.gw.PostDeploy(txReceiptMsg); err != nil {
			log.Warnf("Failed to perform post-deploy processing: %s", err)
			i.ReplyWithReceiptAndError(receipt, err)
			return
		}
	}
	status := 200
	if receipt.ReplyHeaders().MsgType != messages.MsgTypeTransactionSuccess {
		status = 500
	}
	reply, _ := json.MarshalIndent(receipt, "", "  ")
	log.Infof("<-- %s %s [%d]", i.req.Method, i.req.URL, status)
	log.Debugf("<-- %s", reply)
	i.res.Header().Set("Content-Type", "application/json")
	i.res.WriteHeader(status)
	i.res.Write(reply)
	i.done = true
	i.waiter.Broadcast()
	return
}

func newREST2eth(gw SmartContractGateway, cr contractregistry.ContractResolver, rpc eth.RPCClient, subMgr events.SubscriptionManager, processor tx.TxnProcessor, asyncDispatcher REST2EthAsyncDispatcher, syncDispatcher rest2EthSyncDispatcher) *rest2eth {
	return &rest2eth{
		gw:              gw,
		cr:              cr,
		processor:       processor,
		syncDispatcher:  syncDispatcher,
		asyncDispatcher: asyncDispatcher,
		rpc:             rpc,
		subMgr:          subMgr,
	}
}

func (r *rest2eth) addRoutes(router *httprouter.Router) {
	// Built-in registry managed routes
	router.POST("/contracts/:address/:method", r.restHandler)
	router.GET("/contracts/:address/:method", r.restHandler)
	router.POST("/contracts/:address/:method/:subcommand", r.restHandler)

	router.POST("/abis/:abi", r.restHandler)
	router.POST("/abis/:abi/:address/:method", r.restHandler)
	router.GET("/abis/:abi/:address/:method", r.restHandler)
	router.POST("/abis/:abi/:address/:method/:subcommand", r.restHandler)

	// Remote registry managed address routes, with long and short names
	router.POST("/instances/:instance_lookup/:method", r.restHandler)
	router.GET("/instances/:instance_lookup/:method", r.restHandler)
	router.POST("/instances/:instance_lookup/:method/:subcommand", r.restHandler)

	router.POST("/i/:instance_lookup/:method", r.restHandler)
	router.GET("/i/:instance_lookup/:method", r.restHandler)
	router.POST("/i/:instance_lookup/:method/:subcommand", r.restHandler)

	router.POST("/gateways/:gateway_lookup", r.restHandler)
	router.POST("/gateways/:gateway_lookup/:address/:method", r.restHandler)
	router.GET("/gateways/:gateway_lookup/:address/:method", r.restHandler)
	router.POST("/gateways/:gateway_lookup/:address/:method/:subcommand", r.restHandler)

	router.POST("/g/:gateway_lookup", r.restHandler)
	router.POST("/g/:gateway_lookup/:address/:method", r.restHandler)
	router.GET("/g/:gateway_lookup/:address/:method", r.restHandler)
	router.POST("/g/:gateway_lookup/:address/:method/:subcommand", r.restHandler)
}

type restCmd struct {
	from            string
	addr            string
	value           json.Number
	abiLocation     *contractregistry.ABILocation
	abiMethod       *ethbinding.ABIMethod
	abiMethodElem   *ethbinding.ABIElementMarshaling
	abiEvent        *ethbinding.ABIEvent
	abiEventElem    *ethbinding.ABIElementMarshaling
	isDeploy        bool
	deployMsg       *messages.DeployContract
	body            map[string]interface{}
	msgParams       []interface{}
	blocknumber     string
	transactionHash string
}

func (r *rest2eth) resolveABI(res http.ResponseWriter, req *http.Request, params httprouter.Params, c *restCmd, addrParam string) (a ethbinding.ABIMarshaling, validAddress bool, err error) {
	c.addr = strings.ToLower(strings.TrimPrefix(addrParam, "0x"))
	validAddress = addrCheck.MatchString(c.addr)
	var location contractregistry.ABILocation

	// There are multiple ways we resolve the path into an ABI
	// 1. we lookup it up remotely in a REST attached contract registry (the newer option)
	//    - /gateways  is for factory interfaces that can talk to any instance
	//    - /instances is for known individual instances
	// 2. we lookup it up locally in a simple filestore managed in ethconnect (the original option)
	//    - /abis      is for factory interfaces installed into ethconnect by uploading the Solidity
	//    - /contracts is for individual instances deployed via ethconnect factory interfaces
	if strings.HasPrefix(req.URL.Path, "/gateways/") || strings.HasPrefix(req.URL.Path, "/g/") {
		location.ABIType = contractregistry.RemoteGateway
		location.Name = params.ByName("gateway_lookup")
	} else if strings.HasPrefix(req.URL.Path, "/instances/") || strings.HasPrefix(req.URL.Path, "/i/") {
		location.ABIType = contractregistry.RemoteInstance
		location.Name = params.ByName("instance_lookup")
		validAddress = true // assume registry only returns valid addresses
	} else {
		// Local logic
		location.ABIType = contractregistry.LocalABI
		abiID := params.ByName("abi")
		if abiID != "" {
			location.Name = abiID
		} else {
			if !validAddress {
				// Resolve the address as a registered name, to an actual contract address
				if c.addr, err = r.cr.ResolveContractAddress(addrParam); err != nil {
					r.restErrReply(res, req, err, 404)
					return
				}
			}
			validAddress = true
			addrParam = c.addr
			var info *contractregistry.ContractInfo
			if info, err = r.cr.GetContractByAddress(addrParam); err != nil {
				r.restErrReply(res, req, err, 404)
				return
			}
			location.Name = info.ABI
		}
	}

	deployMsg, err := r.cr.GetABI(location, false)
	if err != nil {
		r.restErrReply(res, req, err, 500)
		return
	} else if deployMsg == nil || deployMsg.Contract == nil {
		err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayInstanceNotFound)
		r.restErrReply(res, req, err, 404)
		return
	}
	c.deployMsg = deployMsg.Contract
	c.deployMsg.Headers.ABIID = deployMsg.Contract.Headers.ID // Reference to the original ABI needs to flow through for registration
	c.abiLocation = &location
	if deployMsg.Address != "" {
		c.addr = deployMsg.Address
	}
	a = c.deployMsg.ABI
	return
}

func (r *rest2eth) resolveMethod(res http.ResponseWriter, req *http.Request, c *restCmd, a ethbinding.ABIMarshaling, methodParam string) (err error) {
	for _, element := range a {
		if element.Type == "function" && element.Name == methodParam {
			c.abiMethodElem = &element
			if c.abiMethod, err = ethbind.API.ABIElementMarshalingToABIMethod(&element); err != nil {
				err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMethodABIInvalid, methodParam, err)
				r.restErrReply(res, req, err, 400)
				return
			}
			return
		}
	}
	return
}

func (r *rest2eth) resolveConstructor(res http.ResponseWriter, req *http.Request, c *restCmd, a ethbinding.ABIMarshaling) (err error) {
	for _, element := range a {
		if element.Type == "constructor" {
			c.abiMethodElem = &element
			if c.abiMethod, err = ethbind.API.ABIElementMarshalingToABIMethod(&element); err != nil {
				err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMethodABIInvalid, "constructor", err)
				r.restErrReply(res, req, err, 400)
				return
			}
			c.isDeploy = true
			return
		}
	}
	if !c.isDeploy {
		// Default constructor
		c.abiMethodElem = &ethbinding.ABIElementMarshaling{
			Type: "constructor",
		}
		c.abiMethod, _ = ethbind.API.ABIElementMarshalingToABIMethod(c.abiMethodElem)
		c.isDeploy = true
	}
	return
}

func (r *rest2eth) resolveEvent(res http.ResponseWriter, req *http.Request, c *restCmd, a ethbinding.ABIMarshaling, methodParam, methodParamLC, addrParam string) (err error) {
	var eventDef *ethbinding.ABIElementMarshaling
	for _, element := range a {
		if element.Type == "event" {
			if element.Name == methodParam {
				eventDef = &element
				break
			}
			if methodParamLC == "subscribe" && element.Name == addrParam {
				c.addr = ""
				eventDef = &element
				break
			}
		}
	}
	if eventDef != nil {
		c.abiEventElem = eventDef
		if c.abiEvent, err = ethbind.API.ABIElementMarshalingToABIEvent(eventDef); err != nil {
			err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayEventABIInvalid, eventDef.Name, err)
			r.restErrReply(res, req, err, 400)
			return
		}
	}
	return
}

func (r *rest2eth) resolveParams(res http.ResponseWriter, req *http.Request, params httprouter.Params) (c restCmd, err error) {
	// Check if we have a valid address in :address (verified later if required)
	addrParam := params.ByName("address")
	a, validAddress, err := r.resolveABI(res, req, params, &c, addrParam)
	if err != nil {
		return c, err
	}

	// See addRoutes for all the various routes we support under the factory/instance.
	// We need to handle the special case of
	// /abis/:abi/EVENTNAME/subscribe
	// ... where 'EVENTNAME' is passed as :address and is a valid event
	// and where 'subscribe' is passed as :method

	// Check if we have a method in :method param
	methodParam := params.ByName("method")
	methodParamLC := strings.ToLower(methodParam)
	if methodParam != "" {
		if err = r.resolveMethod(res, req, &c, a, methodParam); err != nil {
			return
		}
	}

	// Then if we don't have a method in :method param, we might have
	// an event in either the :event OR :address param (see special case above)
	// Note solidity guarantees no overlap in method / event names
	if c.abiMethod == nil && methodParam != "" {
		if err = r.resolveEvent(res, req, &c, a, methodParam, methodParamLC, addrParam); err != nil {
			return
		}
	}

	// Last case is the constructor, where nothing is specified
	if methodParam == "" && c.abiMethod == nil && c.abiEvent == nil {
		if err = r.resolveConstructor(res, req, &c, a); err != nil {
			return
		}
	}

	// If we didn't find the method or event, report to the user
	if c.abiMethod == nil && c.abiEvent == nil {
		if methodParamLC == "subscribe" {
			err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayEventNotDeclared, methodParam)
			r.restErrReply(res, req, err, 404)
			return
		}
		err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMethodNotDeclared, url.QueryEscape(methodParam), c.addr)
		r.restErrReply(res, req, err, 404)
		return
	}

	// If we have an address, it must be valid
	if c.addr != "" && !validAddress {
		log.Errorf("Invalid to address: '%s'", params.ByName("address"))
		err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayInvalidToAddress)
		r.restErrReply(res, req, err, 404)
		return
	}
	if c.addr != "" {
		c.addr = "0x" + c.addr
	}

	// If we have a from, it needs to be a valid address
	From := getFlyParam("from", req)
	fromNo0xPrefix := strings.ToLower(strings.TrimPrefix(From, "0x"))
	if fromNo0xPrefix != "" {
		if addrCheck.MatchString(fromNo0xPrefix) {
			c.from = "0x" + fromNo0xPrefix
		} else if tx.IsHDWalletRequest(fromNo0xPrefix) != nil {
			c.from = fromNo0xPrefix
		} else {
			log.Errorf("Invalid from address: '%s'", From)
			err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayInvalidFromAddress)
			r.restErrReply(res, req, err, 404)
			return
		}
	}
	c.value = json.Number(getFlyParam("ethvalue", req))

	c.body, err = utils.YAMLorJSONPayload(req)
	if err != nil {
		r.restErrReply(res, req, err, 400)
		return
	}

	c.blocknumber = getFlyParam("blocknumber", req)
	c.transactionHash = getFlyParam("transaction", req)

	if c.abiEvent != nil || c.transactionHash != "" {
		return
	}

	c.msgParams = make([]interface{}, len(c.abiMethod.Inputs))
	queryParams := req.Form
	for i, abiParam := range c.abiMethod.Inputs {
		argName := abiParam.Name
		// If the ABI input has one or more un-named parameters, look for default names that are passed in.
		// Unnamed Input params should be named: input, input1, input2...
		if argName == "" {
			argName = "input"
			if i != 0 {
				argName += strconv.Itoa(i)
			}
		}
		if bv, exists := c.body[argName]; exists {
			c.msgParams[i] = bv
		} else if vs := queryParams[argName]; len(vs) > 0 {
			c.msgParams[i] = vs[0]
		} else {
			err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMissingParameter, argName, c.abiMethod.Name)
			r.restErrReply(res, req, err, 400)
			return
		}
	}

	return
}

func (r *rest2eth) restHandler(res http.ResponseWriter, req *http.Request, params httprouter.Params) {
	log.Infof("--> %s %s", req.Method, req.URL)

	c, err := r.resolveParams(res, req, params)
	if err != nil {
		return
	}

	if c.abiEvent != nil {
		r.subscribeEvent(res, req, c.addr, c.abiLocation, c.abiEventElem, c.body)
	} else if c.transactionHash != "" {
		r.lookupTransaction(res, req, c.transactionHash, c.abiMethod)
	} else if req.Method != http.MethodPost || c.abiMethod.IsConstant() || getFlyParamBool("call", req) {
		r.callContract(res, req, c.from, c.addr, c.value, c.abiMethod, c.msgParams, c.blocknumber)
	} else {
		if c.from == "" {
			err = ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMissingFromAddress, utils.GetenvOrDefaultLowerCase("PREFIX_SHORT", "fly"), utils.GetenvOrDefaultLowerCase("PREFIX_LONG", "firefly"))
			r.restErrReply(res, req, err, 400)
		} else if c.isDeploy {
			r.deployContract(res, req, c.from, c.value, c.abiMethodElem, c.deployMsg, c.msgParams)
		} else {
			r.sendTransaction(res, req, c.from, c.addr, c.value, c.abiMethodElem, c.msgParams)
		}
	}
}

func (r *rest2eth) fromBodyOrForm(req *http.Request, body map[string]interface{}, param string) string {
	val := body[param]
	valType := reflect.TypeOf(val)
	if valType != nil && valType.Kind() == reflect.String && len(val.(string)) > 0 {
		return val.(string)
	}
	return req.FormValue(param)
}

func (r *rest2eth) subscribeEvent(res http.ResponseWriter, req *http.Request, addrStr string, abi *contractregistry.ABILocation, abiEvent *ethbinding.ABIElementMarshaling, body map[string]interface{}) {

	err := auth.AuthEventStreams(req.Context())
	if err != nil {
		log.Errorf("Unauthorized: %s", err)
		r.restErrReply(res, req, ethconnecterrors.Errorf(ethconnecterrors.Unauthorized), 401)
		return
	}

	if r.subMgr == nil {
		r.restErrReply(res, req, errEventSupportMissing, 405)
		return
	}
	streamID := r.fromBodyOrForm(req, body, "stream")
	if streamID == "" {
		r.restErrReply(res, req, ethconnecterrors.Errorf(ethconnecterrors.RESTGatewaySubscribeMissingStreamParameter), 400)
		return
	}
	fromBlock := r.fromBodyOrForm(req, body, "fromBlock")
	var addr *ethbinding.Address
	if addrStr != "" {
		address := ethbind.API.HexToAddress(addrStr)
		addr = &address
	}
	// if the end user provided a name for the subscription, use it
	// If not provided, it will be set to a system-generated summary
	name := r.fromBodyOrForm(req, body, "name")
	sub, err := r.subMgr.AddSubscription(req.Context(), addr, abi, abiEvent, streamID, fromBlock, name)
	if err != nil {
		r.restErrReply(res, req, err, 400)
		return
	}
	status := 200
	resBytes, _ := json.Marshal(sub)
	log.Infof("<-- %s %s [%d]", req.Method, req.URL, status)
	log.Debugf("<-- %s", resBytes)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(resBytes)
}

func (r *rest2eth) doubleURLDecode(s string) string {
	// Due to an annoying bug in the rapidoc Swagger UI, it is double URL encoding parameters.
	// As most constellation b64 encoded values end in "=" that's breaking the ability to use
	// the UI. As they do not contain a % we just double URL decode them :-(
	// However, this translates '+' into ' ' (space), so we have to fix that too.
	doubleDecoded, _ := url.QueryUnescape(s)
	return strings.ReplaceAll(doubleDecoded, " ", "+")
}

func (r *rest2eth) addPrivateTx(msg *messages.TransactionCommon, req *http.Request, res http.ResponseWriter) error {
	msg.PrivateFrom = r.doubleURLDecode(getFlyParam("privatefrom", req))
	msg.PrivateFor = getFlyParamMulti("privatefor", req)
	for idx, val := range msg.PrivateFor {
		msg.PrivateFor[idx] = r.doubleURLDecode(val)
	}
	msg.PrivacyGroupID = r.doubleURLDecode(getFlyParam("privacygroupid", req))
	if len(msg.PrivateFor) > 0 && msg.PrivacyGroupID != "" {
		return ethconnecterrors.Errorf(ethconnecterrors.RESTGatewayMixedPrivateForAndGroupID, utils.GetenvOrDefaultLowerCase("PREFIX_SHORT", "fly"))
	}
	return nil
}

func (r *rest2eth) assignMessageID(headers *messages.RequestHeaders, req *http.Request) {
	headers.ID = getFlyParam("id", req)
	if headers.ID == "" {
		headers.ID = utils.UUIDv4()
	}
}

func (r *rest2eth) deployContract(res http.ResponseWriter, req *http.Request, from string, value json.Number, abiMethodElem *ethbinding.ABIElementMarshaling, deployMsg *messages.DeployContract, msgParams []interface{}) {

	r.assignMessageID(&deployMsg.Headers, req)
	deployMsg.Headers.MsgType = messages.MsgTypeDeployContract
	deployMsg.From = from
	deployMsg.Gas = json.Number(getFlyParam("gas", req))
	deployMsg.GasPrice = json.Number(getFlyParam("gasprice", req))
	deployMsg.Value = value
	deployMsg.Parameters = msgParams
	if err := r.addPrivateTx(&deployMsg.TransactionCommon, req, res); err != nil {
		r.restErrReply(res, req, err, 400)
		return
	}
	deployMsg.RegisterAs = getFlyParam("register", req)
	if deployMsg.RegisterAs != "" {
		if err := r.cr.CheckNameAvailable(deployMsg.RegisterAs, contractregistry.IsRemote(deployMsg.Headers.CommonHeaders)); err != nil {
			r.restErrReply(res, req, err, 409)
			return
		}
	}
	if getFlyParamBool("sync", req) {
		responder := &rest2EthSyncResponder{
			r:      r,
			res:    res,
			req:    req,
			done:   false,
			waiter: sync.NewCond(&sync.Mutex{}),
		}
		r.syncDispatcher.DispatchDeployContractSync(req.Context(), deployMsg, responder)
		responder.waiter.L.Lock()
		for !responder.done {
			responder.waiter.Wait()
		}
	} else {
		ack := !getFlyParamBool("noack", req) // turn on ack's by default
		immediateReceipt := strings.EqualFold(getFlyParam("acktype", req), "receipt")

		// Async messages are dispatched as generic map payloads.
		// We are confident in the re-serialization here as we've deserialized from JSON then built our own structure
		msgBytes, _ := json.Marshal(deployMsg)
		var mapMsg map[string]interface{}
		json.Unmarshal(msgBytes, &mapMsg)
		if asyncResponse, err := r.asyncDispatcher.DispatchMsgAsync(req.Context(), mapMsg, ack, immediateReceipt); err != nil {
			r.restErrReply(res, req, err, 500)
		} else {
			r.restAsyncReply(res, req, asyncResponse)
		}
	}
	return
}

func (r *rest2eth) sendTransaction(res http.ResponseWriter, req *http.Request, from, addr string, value json.Number, abiMethodElem *ethbinding.ABIElementMarshaling, msgParams []interface{}) {

	msg := &messages.SendTransaction{}
	r.assignMessageID(&msg.Headers, req)
	msg.Headers.MsgType = messages.MsgTypeSendTransaction
	msg.Method = abiMethodElem
	msg.To = addr
	msg.From = from
	msg.Gas = json.Number(getFlyParam("gas", req))
	msg.GasPrice = json.Number(getFlyParam("gasprice", req))
	msg.Value = value
	msg.Parameters = msgParams
	if err := r.addPrivateTx(&msg.TransactionCommon, req, res); err != nil {
		r.restErrReply(res, req, err, 400)
		return
	}

	if getFlyParamBool("sync", req) {
		responder := &rest2EthSyncResponder{
			r:      r,
			res:    res,
			req:    req,
			done:   false,
			waiter: sync.NewCond(&sync.Mutex{}),
		}
		r.syncDispatcher.DispatchSendTransactionSync(req.Context(), msg, responder)
		responder.waiter.L.Lock()
		for !responder.done {
			responder.waiter.Wait()
		}
	} else {
		ack := !getFlyParamBool("noack", req) // turn on ack's by default
		immediateReceipt := strings.EqualFold(getFlyParam("acktype", req), "receipt")

		// Async messages are dispatched as generic map payloads.
		// We are confident in the re-serialization here as we've deserialized from JSON then built our own structure
		msgBytes, _ := json.Marshal(msg)
		var mapMsg map[string]interface{}
		json.Unmarshal(msgBytes, &mapMsg)
		if asyncResponse, err := r.asyncDispatcher.DispatchMsgAsync(req.Context(), mapMsg, ack, immediateReceipt); err != nil {
			r.restErrReply(res, req, err, 500)
		} else {
			r.restAsyncReply(res, req, asyncResponse)
		}
	}
	return
}

func (r *rest2eth) callContract(res http.ResponseWriter, req *http.Request, from, addr string, value json.Number, abiMethod *ethbinding.ABIMethod, msgParams []interface{}, blocknumber string) {
	var err error
	if from, err = r.processor.ResolveAddress(from); err != nil {
		r.restErrReply(res, req, err, 500)
		return
	}

	resBody, err := eth.CallMethod(req.Context(), r.rpc, nil, from, addr, value, abiMethod, msgParams, blocknumber)
	if err != nil {
		r.restErrReply(res, req, err, 500)
		return
	}
	resBytes, _ := json.MarshalIndent(&resBody, "", "  ")
	status := 200
	log.Infof("<-- %s %s [%d]", req.Method, req.URL, status)
	log.Debugf("<-- %s", resBytes)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(resBytes)
	return
}

func (r *rest2eth) lookupTransaction(res http.ResponseWriter, req *http.Request, txHash string, abiMethod *ethbinding.ABIMethod) {
	info, err := eth.GetTransactionInfo(req.Context(), r.rpc, txHash)
	if err != nil {
		r.restErrReply(res, req, err, 500)
		return
	}
	inputArgs, err := eth.DecodeInputs(abiMethod, info.Input)
	if err != nil {
		r.restErrReply(res, req, err, 500)
		return
	}

	resBody := messages.TransactionInfo{
		BlockHash:           info.BlockHash,
		BlockNumberHex:      info.BlockNumber,
		From:                info.From,
		To:                  info.To,
		GasHex:              info.Gas,
		GasPriceHex:         info.GasPrice,
		Hash:                info.Hash,
		NonceHex:            info.Nonce,
		TransactionIndexHex: info.TransactionIndex,
		ValueHex:            info.Value,
		Input:               info.Input,
		InputArgs:           inputArgs,
	}

	if info.BlockNumber != nil {
		resBody.BlockNumberStr = info.BlockNumber.ToInt().Text(10)
	}
	if info.Gas != nil {
		resBody.GasStr = strconv.FormatUint(uint64(*info.Gas), 10)
	}
	if info.GasPrice != nil {
		resBody.GasPriceStr = info.GasPrice.ToInt().Text(10)
	}
	if info.Nonce != nil {
		resBody.NonceStr = strconv.FormatUint(uint64(*info.Nonce), 10)
	}
	if info.TransactionIndex != nil {
		resBody.TransactionIndexStr = strconv.FormatUint(uint64(*info.TransactionIndex), 10)
	}
	if info.Value != nil {
		resBody.ValueStr = info.Value.ToInt().Text(10)
	}

	resBytes, _ := json.MarshalIndent(&resBody, "", "  ")
	status := 200
	log.Infof("<-- %s %s [%d]", req.Method, req.URL, status)
	log.Debugf("<-- %s", resBytes)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(resBytes)
	return
}

func (r *rest2eth) restAsyncReply(res http.ResponseWriter, req *http.Request, asyncResponse *messages.AsyncSentMsg) {
	resBytes, _ := json.Marshal(asyncResponse)
	status := 202 // accepted
	log.Infof("<-- %s %s [%d]:\n%s", req.Method, req.URL, status, string(resBytes))
	log.Debugf("<-- %s", resBytes)
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(resBytes)
}

func (r *rest2eth) restErrReply(res http.ResponseWriter, req *http.Request, err error, status int) {
	log.Errorf("<-- %s %s [%d]: %s", req.Method, req.URL, status, err)
	reply, _ := json.Marshal(errors.ToRESTError(err))
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	res.Write(reply)
	return
}
