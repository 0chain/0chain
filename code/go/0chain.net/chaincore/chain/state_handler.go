package chain

import (
	"context"
	"encoding/json"
	"net/http"
	"regexp"

	"0chain.net/chaincore/smartcontract"

	"0chain.net/chaincore/smartcontractstate"

	"0chain.net/core/common"
)

/*SetupStateHandlers - setup handlers to manage state */
func SetupStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.UserRateLimit(common.ToJSONResponse(c.GetBalanceHandler)))
	http.HandleFunc("/v1/scstate/get", common.UserRateLimit(common.ToJSONResponse(c.GetNodeFromSCState)))
	http.HandleFunc("/v1/screst/", common.UserRateLimit(common.ToJSONResponse(c.GetSCRestOutput)))
}

func (c *Chain) GetSCRestOutput(ctx context.Context, r *http.Request) (interface{}, error) {
	scRestRE := regexp.MustCompile(`/v1/screst/(.*)?/(.*)`)
	pathParams := scRestRE.FindStringSubmatch(r.URL.Path)
	if len(pathParams) < 3 {
		return nil, common.NewError("invalid_path", "Invalid Rest API path")
	}

	scAddress := pathParams[1]
	scRestPath := "/" + pathParams[2]

	mndb := smartcontractstate.NewMemorySCDB()
	ndb := smartcontractstate.NewPipedSCDB(mndb, c.scStateDB, false)

	resp, err := smartcontract.ExecuteRestAPI(ctx, scAddress, scRestPath, r.URL.Query(), ndb)

	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (c *Chain) GetNodeFromSCState(ctx context.Context, r *http.Request) (interface{}, error) {
	scAddress := r.FormValue("sc_address")
	key := r.FormValue("key")
	pdb := c.scStateDB
	scState := smartcontractstate.NewSCState(pdb, scAddress)
	node, err := scState.GetNode(smartcontractstate.Key(key))
	if err != nil {
		return nil, err
	}
	if node == nil {
		return nil, common.NewError("key_not_found", "key was not found")
	}
	var retObj interface{}
	err = json.Unmarshal(node, &retObj)
	if err != nil {
		return nil, err
	}
	return retObj, nil
}

/*GetBalanceHandler - get the balance of a client */
func (c *Chain) GetBalanceHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	clientID := r.FormValue("client_id")
	lfb := c.LatestFinalizedBlock
	if lfb == nil {
		return nil, common.ErrTemporaryFailure
	}
	state, err := c.GetState(lfb, clientID)
	if err != nil {
		return nil, err
	}
	return state, nil
}
