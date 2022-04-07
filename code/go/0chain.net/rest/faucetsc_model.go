package rest

import (
	"time"

	"0chain.net/chaincore/state"
)

// swagger:model periodicResponse
type periodicResponse struct {
	Used    state.Balance `json:"tokens_poured"`
	Start   time.Time     `json:"start_time"`
	Restart string        `json:"time_left"`
	Allowed state.Balance `json:"tokens_allowed"`
}
