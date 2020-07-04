package smartcontractinterface

import (
	"encoding/json"

	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
)

type PoolStats struct {
	DelegateID   string        `json:"delegate_id"`
	High         state.Balance `json:"high"` // } interests and rewards
	Low          state.Balance `json:"low"`  // }
	InterestPaid state.Balance `json:"interest_paid"`
	RewardPaid   state.Balance `json:"reward_paid"`
	NumRounds    int64         `json:"number_rounds"`
	Status       string        `json:"status"`
}

func (ps *PoolStats) AddInterests(value state.Balance) {
	ps.InterestPaid += value
	if ps.Low < 0 {
		ps.Low = value
	} else if value < ps.Low {
		ps.Low = value
	}
	if value > ps.High {
		ps.High = value
	}
}

func (ps *PoolStats) AddRewards(value state.Balance) {
	ps.RewardPaid += value
	if ps.Low < 0 {
		ps.Low = value
	} else if value < ps.Low {
		ps.Low = value
	}
	if value > ps.High {
		ps.High = value
	}
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
