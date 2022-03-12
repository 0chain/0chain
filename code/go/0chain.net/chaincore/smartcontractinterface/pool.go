package smartcontractinterface

/*
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

*/
