package restinterface

import (
	"0chain.net/chaincore/chain/state"
)

type RestHandlerI interface {
	GetStateContext() state.QueryStateContextI
	SetupRestHandlers()
	SetQueryStateContext(state.QueryStateContextI)
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

func (rh *TestRestHandler) GetStateContext() state.QueryStateContextI {
	return rh.GetQueryStateContext()
}

func (rh *TestRestHandler) SetupRestHandlers() {
}
