package rest

import (
	"0chain.net/chaincore/chain/state"
	"net/http"
)

type Endpoint struct {
	URI     string
	Handler func(w http.ResponseWriter, r *http.Request)
}

func MakeEndpoint(uri string, f func(w http.ResponseWriter, r *http.Request)) Endpoint {
	return Endpoint{
		URI:     uri,
		Handler: f,
	}
}

// swagger:model Int64Map
type Int64Map map[string]int64

// swagger:model InterfaceMap
type InterfaceMap map[string]interface{}

type QueryChainer interface {
	GetQueryStateContext() state.QueryStateContextI
	SetQueryStateContext(state.QueryStateContextI)
}

type RestHandlerI interface {
	QueryChainer
	Register([]Endpoint)
}

type TestQueryChainer struct {
	sctx state.QueryStateContextI
}

func (qc *TestQueryChainer) GetQueryStateContext() state.QueryStateContextI {
	return qc.sctx
}

func (qc *TestQueryChainer) SetQueryStateContext(sctx state.QueryStateContextI) {
	qc.sctx = sctx
}

type RestHandler struct {
	QueryChainer
}

func NewRestHandler(c QueryChainer) RestHandlerI {
	return &RestHandler{QueryChainer: c}
}

func (rh *RestHandler) Register(endpoints []Endpoint) {
	for _, e := range endpoints {
		http.HandleFunc(e.URI, e.Handler)
	}
}
