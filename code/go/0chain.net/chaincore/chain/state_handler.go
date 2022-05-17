package chain

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"0chain.net/core/logging"

	"go.uber.org/zap"

	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
	"github.com/tinylib/msgp/msgp"
)

/*SetupStateHandlers - setup handlers to manage state */
func SetupStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.UserRateLimit(common.ToJSONResponse(c.GetBalanceHandler)))
	http.HandleFunc("/v1/scstate/get", common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState)))
	http.HandleFunc("/v1/scstats/", common.UserRateLimit(c.GetSCStats))
	http.HandleFunc("/v1/screst/", common.UserRateLimit(c.HandleSCRest))
	http.HandleFunc("/_smart_contract_stats", common.UserRateLimit(c.SCStats))
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
			common.ToJSONResponse(c.GetSCRestOutput)(w, r)
		} else {
			c.GetSCRestPoints(w, r)
		}
	}
}

func (c *Chain) GetSCRestOutput(ctx context.Context, r *http.Request) (interface{}, error) {
	scRestRE := regexp.MustCompile(`/v1/screst/(.*)?/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 3 {
		return nil, common.NewError("invalid_path", "Invalid Rest API path")
	}

	scAddress := pathParams[1]
	scRestPath := "/" + pathParams[2]
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()

	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil || lfb.ClientState == nil {
		return nil, common.NewError("empty_lfb", "empty latest finalized block or state")
	}
	clientState := CreateTxnMPT(lfb.ClientState) // begin transaction
	sctx := c.NewStateContext(lfb, clientState, &transaction.Transaction{}, c.GetEventDb())
	resp, err := smartcontract.ExecuteRestAPI(ctx, scAddress, scRestPath, r.URL.Query(), sctx)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Chain) GetNodeFromSCState(ctx context.Context, r *http.Request) (interface{}, error) {
	scAddress := r.FormValue("sc_address")
	key := r.FormValue("key")
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

/*GetBalanceHandler - get the balance of a client */
func (c *Chain) GetBalanceHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	clientID := r.FormValue("client_id")
	lfb := c.GetLatestFinalizedBlock()
	if lfb == nil {
		return nil, common.ErrTemporaryFailure
	}
	state, err := c.GetState(lfb, clientID)
	logging.Logger.Info("piers GetBalanceHandler",
		zap.String("client_id", clientID),
		zap.Any("state", state))
	if err != nil {
		return nil, err
	}
	if err := state.ComputeProperties(); err != nil {
		return nil, err
	}
	logging.Logger.Info("piers GetBalanceHandler end", zap.Any("state", state), zap.String("clientID", clientID))
	return state, nil
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
	fmt.Fprintf(w, "<tr class='header'><td>Type</td><td>ID</td><td>Link</td><td>RestAPIs</td></tr>")
	re := regexp.MustCompile(`\*.*\.`)
	keys := make([]string, 0, len(smartcontract.ContractMap))
	for k := range smartcontract.ContractMap {
		keys = append(keys, k)
	}
	sort.SliceStable(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		sc := smartcontract.ContractMap[k]
		scType := re.ReplaceAllString(reflect.TypeOf(sc).String(), "")
		fmt.Fprintf(w, `<tr><td>%v</td><td>%v</td><td><li><a href='%v'>%v</a></li></td><td><li><a href='%v'>%v</a></li></td></tr>`, scType, strings.ToLower(k), "v1/scstats/"+k, "/v1/scstats/"+scType, "v1/screst/"+k, "/v1/screst/*key*")
	}
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) GetSCRestPoints(w http.ResponseWriter, r *http.Request) {
	scRestRE := regexp.MustCompile(`/v1/screst/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 2 {
		return
	}
	key := pathParams[1]
	scInt, ok := smartcontract.ContractMap[key]
	if !ok {
		return
	}
	PrintCSS(w)
	fmt.Fprintf(w, "<table class='menu' style='border-collapse: collapse;'>")
	fmt.Fprintf(w, "<tr class='header'><td>Function</td><td>Link</td></tr>")
	restPoints := scInt.GetRestPoints()
	names := make([]string, 0, len(restPoints))
	for funcName := range restPoints {
		names = append(names, funcName)
	}
	sort.Strings(names)
	for _, funcName := range names {
		friendlyName := strings.TrimLeft(funcName, "/")
		fmt.Fprintf(w, `<tr><td>%v</td><td><li><a href='%v'>%v</a></li></td></tr>`, friendlyName, key+funcName, "/v1/screst/*"+funcName+"*")
	}
	fmt.Fprintf(w, "</table>")
}
