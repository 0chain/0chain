package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/rest"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/statecache"
	"github.com/0chain/common/core/util"
	"github.com/go-openapi/runtime/middleware"
	"github.com/tinylib/msgp/msgp"
)

func SetupSwagger() {
	http.Handle("/swagger.yaml", http.FileServer(http.Dir("/docs")))

	// documentation for developers
	opts := middleware.SwaggerUIOpts{SpecURL: "swagger.yaml"}
	sh := middleware.SwaggerUI(opts, nil)
	http.Handle("/docs", sh)

	// documentation for share
	opts1 := middleware.RedocOpts{SpecURL: "swagger.yaml", Path: "docs1"}
	sh1 := middleware.Redoc(opts1, nil)
	http.Handle("/docs1", sh1)
}

func SetupScRestApiHandlers() {
	c := GetServerChain()
	restHandler := rest.NewRestHandler(c)
	SetupSwagger()
	if c.EventDb != nil {
		faucetsc.SetupRestHandler(restHandler)
		minersc.SetupRestHandler(restHandler)
		storagesc.SetupRestHandler(restHandler)
		vestingsc.SetupRestHandler(restHandler)
		zcnsc.SetupRestHandler(restHandler)

	} else {
		logging.Logger.Warn("cannot find event database, REST API will not be supported on this sharder")
	}
}

/*SetupStateHandlers - setup sharder handlers to manage state */
func SetupSharderStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.WithCORS(common.UserRateLimit(common.ToJSONResponse(c.GetBalanceHandler))))
	http.HandleFunc("/v1/current-round", common.WithCORS(common.UserRateLimit(common.ToJSONResponse(c.GetCurrentRoundHandler))))
	http.HandleFunc("/v1/scstats/", common.WithCORS(common.UserRateLimit(c.GetSCStats)))
	http.HandleFunc("/v1/screst/", common.WithCORS(common.UserRateLimit(c.HandleSCRest)))
}

/*SetupStateHandlers - setup handlers to manage state */
func SetupDebugStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/scstate/get", common.WithCORS(common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState))))
}

func SetupStateHandlers() {
	http.HandleFunc("/_smart_contract_stats", common.WithCORS(common.UserRateLimit(GetServerChain().SCStats)))
}

func (c *Chain) GetQueryStateContext() state.TimedQueryStateContextI {
	return state.NewTimedQueryStateContext(c.GetStateContextI(), func() common.Timestamp {
		return common.Now()
	})
}

func (c *Chain) SetQueryStateContext(_ state.TimedQueryStateContextI) {
}

func (c *Chain) GetStateContext() state.StateContextI {
	return c.GetStateContextI()
}

func (c *Chain) GetStateContextI() state.StateContextI {
	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil || lfb.ClientState == nil {
		logging.Logger.Error("empty latest finalized block or state")
		return nil
	}
	qbc := statecache.NewQueryBlockCache(c.GetStateCache(), lfb.Hash)
	tbc := statecache.NewTransactionCache(qbc)
	clientState := CreateTxnMPT(lfb.ClientState, tbc) // begin transaction
	return c.NewStateContext(lfb, clientState, &transaction.Transaction{}, c.GetEventDb())
}

func (c *Chain) HandleSCRest(w http.ResponseWriter, r *http.Request) {
	scRestRE := regexp.MustCompile(`/v1/screst/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 2 {
		return
	}

	if len(pathParams) == 2 {
		scRestRE = regexp.MustCompile(`/v1/screst/(.*)?/(.*)`)
		pathParams = scRestRE.FindStringSubmatch(r.URL.Path)
		if len(pathParams) == 3 {
			// This is a call for an undefined endpoint, it's undefined since it fell back to this handler instead of the actual handler of the endpoint
			fmt.Fprintf(w, "invalid_path: Invalid Rest API path")
			w.WriteHeader(http.StatusNotFound)
			return
		} else {
			c.GetSCRestPoints(w, r)
		}
	}
}

func (c *Chain) GetNodeFromSCState(ctx context.Context, r *http.Request) (interface{}, error) {
	scAddress := r.FormValue("sc_address")
	key := r.FormValue("key")
	block := r.FormValue("block")
	if len(block) > 0 {
		b, err := c.GetBlock(ctx, block)
		if err != nil {
			return nil, err
		}

		if b.ClientState == nil {
			return nil, errors.New("block client state is nil")
		}

		d, err := b.ClientState.GetNodeValueRaw(util.Path(encryption.Hash(scAddress + key)))
		if err != nil {
			return nil, err
		}
		if len(d) == 0 {
			return nil, common.NewError("key_not_found", "key was not found")
		}

		buf := &bytes.Buffer{}
		_, err = msgp.UnmarshalAsJSON(buf, d)
		if err != nil {
			return nil, common.NewErrorf("decode error", "unmarshal as json failed: %v", err)
		}

		var retObj interface{}
		err = json.NewDecoder(buf).Decode(&retObj)
		if err != nil {
			return nil, err
		}
		return retObj, nil
	}

	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil {
		return nil, common.NewError("failed to get sc state", "finalized block doesn't exist")
	}
	if lfb.ClientState == nil {
		return nil, common.NewError("failed to get sc state", "finalized block's state doesn't exist")
	}
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	d, err := lfb.ClientState.GetNodeValueRaw(util.Path(encryption.Hash(scAddress + key)))
	if err != nil {
		return nil, err
	}
	if len(d) == 0 {
		return nil, common.NewError("key_not_found", "key was not found")
	}

	buf := &bytes.Buffer{}
	_, err = msgp.UnmarshalAsJSON(buf, d)
	if err != nil {
		return nil, common.NewErrorf("decode error", "unmarshal as json failed: %v", err)
	}

	var retObj interface{}
	err = json.NewDecoder(buf).Decode(&retObj)
	if err != nil {
		return nil, err
	}
	return retObj, nil
}

// GetBalanceHandler - get the balance of a client
// swagger:route GET /v1/client/get/balance sharder GetClientBalance
// Get client balance.
// Retrieves the balance of a client.
//
// parameters:
//
//	+name: client_id
//	  in: query
//	  required: true
//	  type: string
//	  description: Client ID
//
// responses:
//
//	200: State
//	400:
func (c *Chain) GetBalanceHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	clientID := r.FormValue("client_id")
	if c.GetEventDb() == nil {
		return nil, common.NewError("get_balance_error", "event database not enabled")
	}

	user, err := c.GetEventDb().GetUser(clientID)
	if err != nil {
		return nil, err
	}

	return userToState(user), nil
}

// swagger:route GET /v1/current-round sharder GetCurrentRound
// Get round.
// Retrieves the current round number as int64.
//
// Responses:
//
//	200:
//	400:
func (c *Chain) GetCurrentRoundHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return c.GetCurrentRound(), nil
}

func (c *Chain) GetSCStats(w http.ResponseWriter, r *http.Request) {
	scRestRE := regexp.MustCompile(`/v1/scstats/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 2 {
		fmt.Fprintf(w, "invalid_path: Invalid Rest API path")
		return
	}
	ctx := common.GetRootContext()
	scAddress := pathParams[1]

	w.Header().Set("Content-Type", "text/html")
	PrintCSS(w)
	smartcontract.ExecuteStats(ctx, scAddress, r.URL.Query(), w)
}

func (c *Chain) SCStats(w http.ResponseWriter, r *http.Request) {
	PrintCSS(w)
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'> <td>Type</td><td>ID</td><td>Link</td></tr>")
	re := regexp.MustCompile(`\*.*\.`)
	keys := make([]string, 0, len(smartcontract.ContractMap))
	for k := range smartcontract.ContractMap {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		sc := smartcontract.ContractMap[k]
		scType := re.ReplaceAllString(reflect.TypeOf(sc).String(), "")
		fmt.Fprintf(w, `<tr><td>%v</td><td>%v</td><td><li><a href='%v'>%v</a></li></td></tr>`, scType, strings.ToLower(k), "v1/scstats/"+k, "/v1/scstats/"+scType)
	}
	fmt.Fprintf(w, "</table>")
}

func GetFunctionNames(address string) []string {
	var endpoints []rest.Endpoint
	switch address {
	case storagesc.ADDRESS:
		endpoints = storagesc.GetEndpoints(nil)
	case minersc.ADDRESS:
		endpoints = minersc.GetEndpoints(nil)
	case faucetsc.ADDRESS:
		endpoints = faucetsc.GetEndpoints(nil)
	case vestingsc.ADDRESS:
		endpoints = vestingsc.GetEndpoints(nil)
	case zcnsc.ADDRESS:
		endpoints = zcnsc.GetEndpoints(nil)
	default:
		return []string{}
	}
	var names []string
	for _, endpoint := range endpoints {
		names = append(names, endpoint.URI)
	}
	return names
}

func (c *Chain) GetSCRestPoints(w http.ResponseWriter, r *http.Request) {
	scRestRE := regexp.MustCompile(`/v1/screst/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 2 {
		return
	}

	PrintCSS(w)
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Function</td><td>Link</td></tr>")

	key := pathParams[1]                     // same as the smart contract adress
	names := GetFunctionNames(pathParams[1]) // fill link of endpoint: /v1/screst/ADDRESS/getAuthorizer

	sort.Strings(names)
	for _, funcName := range names {
		friendlyName := strings.TrimLeft(funcName, "/")
		paths := strings.Split(funcName, "/")
		route := "/" + paths[len(paths)-1]
		fmt.Fprintf(w, `<tr><td>%v</td><td><li><a href='%v'>%v</a></li></td></tr>`, friendlyName, key+route, "/v1/screst/*"+funcName+"*")
	}
	fmt.Fprintf(w, "</table>")
}
