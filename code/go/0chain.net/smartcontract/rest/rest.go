package rest

import (
	"net/http"

	"0chain.net/chaincore/chain/state"
	"github.com/0chain/common/core/util"
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
	GetQueryStateContext() state.TimedQueryStateContextI
	GetStateContextI() state.StateContextI
	SetQueryStateContext(state.TimedQueryStateContextI)
}

type RestHandlerI interface {
	QueryChainer
	Register([]Endpoint)
}

type TestQueryChainer struct {
	sctx state.TimedQueryStateContextI
}

func (qc *TestQueryChainer) GetQueryStateContext() state.TimedQueryStateContextI {
	return qc.sctx
}

func (qc *TestQueryChainer) SetQueryStateContext(sctx state.TimedQueryStateContextI) {
	qc.sctx = sctx
}

func CreateTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot())
	return tmpt
}

func (qc *TestQueryChainer) GetStateContextI() state.StateContextI {
	lfb := qc.sctx.GetLatestFinalizedBlock()
	if lfb == nil || lfb.ClientState == nil {
		return nil
	}
	clientState := CreateTxnMPT(lfb.ClientState) // begin transaction
	s := qc.sctx
	return state.NewStateContext(
		s.GetBlock(),
		clientState,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		s.GetEventDB())
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
