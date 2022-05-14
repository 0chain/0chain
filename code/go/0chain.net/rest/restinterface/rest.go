package restinterface

import (
	"0chain.net/chaincore/chain/state"
)

type StateContextAccessor interface {
	GetROStateContext() state.QueryStateContextI
	GetCurrentRound() int64
}

type RestHandlerI interface {
	GetSC() state.QueryStateContextI
	SetScAccessor(StateContextAccessor)
	SetupRestHandlers()
}

// swagger:model Int64Map
type Int64Map map[string]int64

// swagger:model InterfaceMap
type InterfaceMap map[string]interface{}
