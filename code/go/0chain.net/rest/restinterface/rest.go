package restinterface

import (
	"0chain.net/chaincore/chain/state"
)

type RestHandlerI interface {
	state.ReadOnlyStateContextI
	SetupRestHandlers()
	SetStateContext(state.ReadOnlyStateContextI)
}

// swagger:model Int64Map
type Int64Map map[string]int64

// swagger:model InterfaceMap
type InterfaceMap map[string]interface{}
