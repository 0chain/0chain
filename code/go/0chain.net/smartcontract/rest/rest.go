package rest

import (
	"0chain.net/smartcontract/common"
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
	GetQueryStateContext() common.TimedQueryStateContextI
	SetQueryStateContext(common.TimedQueryStateContextI)
}

type RestHandlerI interface {
	QueryChainer
	Register([]Endpoint)
}

type TestQueryChainer struct {
	sctx common.TimedQueryStateContextI
}

func (qc *TestQueryChainer) GetQueryStateContext() common.TimedQueryStateContextI {
	return qc.sctx
}

func (qc *TestQueryChainer) SetQueryStateContext(sctx common.TimedQueryStateContextI) {
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
		http.HandleFunc(e.URI, WithCORS(e.Handler))
	}
}

// WithCORS enable CORS
func WithCORS(fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") // CORS for all.
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.Header().Add("Access-Control-Max-Age", "3600")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		fn(w, r)
	}
}
