package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
)

type PoolStats struct {
	DelegateID   string        `json:"delegate_id"`
	High         state.Balance `json:"high"`
	Low          state.Balance `json:"low"`
	InterestRate float64       `json:"interest_rate"`
	TotalPaid    state.Balance `json:"total_paid"`
	NumRounds    int64         `json:"number_rounds"`
	Status       string        `json:"status"`
}

func (ps *PoolStats) Encode() []byte {
	buff, _ := json.Marshal(ps)
	return buff
}

func (ps *PoolStats) Decode(input []byte) error {
	return json.Unmarshal(input, ps)
}

type DelegatePool struct {
	*PoolStats                `json:"stats"`
	*tokenpool.ZcnLockingPool `json:"pool"`
}

func NewDelegatePool() *DelegatePool {
	return &DelegatePool{ZcnLockingPool: &tokenpool.ZcnLockingPool{}, PoolStats: &PoolStats{Low: -1}}
}

func (dp *DelegatePool) Encode() []byte {
	buff, _ := json.Marshal(dp)
	return buff
}

func (dp *DelegatePool) Decode(input []byte, tokenlock tokenpool.TokenLockInterface) error {
	var objMap map[string]*json.RawMessage
	err := json.Unmarshal(input, &objMap)
	if err != nil {
		return err
	}
	ps, ok := objMap["stats"]
	if ok {
		var stats *PoolStats
		err = json.Unmarshal(*ps, &stats)
		if err != nil {
			return err
		}
		dp.PoolStats = stats
	}
	p, ok := objMap["pool"]
	if ok {
		err = dp.ZcnLockingPool.Decode(*p, tokenlock)
		if err != nil {
			return err
		}
	}
	return nil
}
