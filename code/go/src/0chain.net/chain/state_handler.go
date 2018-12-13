package chain

import (
	"context"
	"net/http"

	"0chain.net/smartcontractstate"

	"0chain.net/common"
)

/*SetupStateHandlers - setup handlers to manage state */
func SetupStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.ToJSONResponse(c.GetBalanceHandler))
	http.HandleFunc("/v1/scstate/get", common.ToJSONResponse(c.GetNodeFromSCState))
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
	return string(node), nil
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
