package rest

import (
	"0chain.net/chaincore/chain/state"
	"net/http"
)

type RestEndpoint struct {
	Name     string
	Endpoint func(w http.ResponseWriter, r *http.Request)
}

type RestHandlerI interface {
	QueryChainer
	Register([]RestEndpoint)
}

// swagger:model Int64Map
type Int64Map map[string]int64

// swagger:model InterfaceMap
type InterfaceMap map[string]interface{}

type QueryChainer interface {
	GetQueryStateContext() state.QueryStateContextI
	SetQueryStateContext(state.QueryStateContextI)
}

type queryChainer struct {
	sctx state.QueryStateContextI
}

func (qc *queryChainer) GetQueryStateContext() state.QueryStateContextI {
	return qc.sctx
}

func (qc *queryChainer) SetQueryStateContext(sctx state.QueryStateContextI) {
	qc.sctx = sctx
}

type TestRestHandler struct {
	QueryChainer
}

func NewTestRestHandler() RestHandlerI {
	return &TestRestHandler{
		QueryChainer: &queryChainer{},
	}
}

func (rh *TestRestHandler) Register(endpoints []RestEndpoint) {
	for _, e := range endpoints {
		http.HandleFunc(e.Name, e.Endpoint)
	}
}

func (rh *TestRestHandler) SetupRestHandlers() {
}
