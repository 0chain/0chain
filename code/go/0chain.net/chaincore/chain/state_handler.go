package chain

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strings"

	bcstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"

	"0chain.net/chaincore/transaction"
	// "0chain.net/smartcontract/faucetsc"

	"0chain.net/core/common"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

/*SetupStateHandlers - setup handlers to manage state */
func SetupStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.UserRateLimit(common.ToJSONResponse(c.GetBalanceHandler)))
	http.HandleFunc("/v1/scstate/get", common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState)))
	http.HandleFunc("/v1/screst/", common.UserRateLimit(common.ToJSONResponse(c.GetSCRestOutput)))
	http.HandleFunc("/v1/scstats/", common.UserRateLimit(c.GetSCStats))
	http.HandleFunc("/v1/scrests/", common.UserRateLimit(c.GetSCRestPoints))
	http.HandleFunc("/_smart_contract_stats", common.UserRateLimit(c.SCStats))
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
	clientState := createTxnMPT(lfb.ClientState) // begin transaction
	txn := &transaction.Transaction{}
	sctx := bcstate.NewStateContext(lfb, clientState, c.clientStateDeserializer, txn, c.GetBlockSharders)
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
	node, err := lfb.ClientState.GetNodeValue(util.Path(encryption.Hash(scAddress + key)))
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, common.NewError("key_not_found", "key was not found")
	}
	var retObj interface{}
	err = json.Unmarshal(node.Encode(), &retObj)
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
	if err != nil {
		return nil, err
	}
	state.ComputeProperties()
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
		fmt.Fprintf(w, `<tr><td>%v</td><td>%v</td><td><li><a href='%v'>%v</a></li></td><td><li><a href='%v'>%v</a></li></td></tr>`, scType, strings.ToLower(k), "/v1/scstats/"+k, "/v1/scstats/"+scType, "/v1/scrests/"+k, "/v1/scrests/*key*")
	}
	fmt.Fprintf(w, "</table>")
}

func (c *Chain) GetSCRestPoints(w http.ResponseWriter, r *http.Request) {
	scRestRE := regexp.MustCompile(`/v1/scrests/(.*)`)
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
	sc := sci.NewSC(key)
	scInt.SetSC(sc, nil)
	fmt.Fprintf(w, `<!DOCTYPE html><html><body><table class='menu' style='border-collapse: collapse;'>`)
	fmt.Fprintf(w, `<tr class='header'><td>Function</td><td>Link</td></tr>`)
	for funcName := range scInt.GetRestPoints() {
		fmt.Fprintf(w, `<tr><td>%v</td><td><li><a href='%v'>%v</a></li></td></tr>`, funcName, "/v1/screst/"+key+funcName, "/v1/screst/*"+funcName+"*")
	}

	fmt.Fprintf(w, `</table></body></html>`)
}
