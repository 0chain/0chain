package chain

import (
	"context"
	"net/http"

	"0chain.net/common"
)

/*SetupStateHandlers - setup handlers to manage state */
func SetupStateHandlers() {
	c := GetServerChain()
	http.HandleFunc("/v1/client/get/balance", common.ToJSONResponse(c.GetBalanceHandler))
}

/*GetBalanceHandler - get the balance of a client */
func (c *Chain) GetBalanceHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	clientID := r.FormValue("client_id")
	lfb := c.LatestFinalizedBlock
	if lfb == nil {
		return nil, common.ErrTemporaryFailure
	}
	balance, err := c.GetState(lfb, clientID)
	if err != nil {
		return nil, err
	}
	sr := &StateResponse{Round: lfb.Round, BlockHash: lfb.Hash}
	sr.State = balance
	return sr, nil
}
