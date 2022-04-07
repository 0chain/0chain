package restinterface

import (
	"0chain.net/chaincore/chain/state"
)

type RestHandlerI interface {
	state.ReadOnlyStateContextI
	SetupRestHandlers()
	SetStateContext(state.StateContextI)
}
