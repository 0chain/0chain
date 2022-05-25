package rest

import (
	"0chain.net/chaincore/chain/state"
	"net/http"
)

type RestHandler struct {
	QueryChainer
}

func NewRestHandler(c QueryChainer) RestHandlerI {
	return &RestHandler{QueryChainer: c}
}

func (rh *RestHandler) GetQueryStateContext() state.QueryStateContextI {
	return rh.GetQueryStateContext()
}

func (rh *RestHandler) Register(endpoints []RestEndpoint) {
	for _, e := range endpoints {
		http.HandleFunc(e.Name, e.Endpoint)
	}
}
