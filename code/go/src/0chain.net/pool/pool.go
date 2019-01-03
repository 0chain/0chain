package pool

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/state"
)

type Pool struct {
	ID      string        `json:"id"`
	Balance state.Balance `json:"amount"`
}

func (p *Pool) Encode() []byte {
	buff, _ := json.Marshal(p)
	return buff
}

func (p *Pool) Decode(input []byte) error {
	err := json.Unmarshal(input, p)
	return err
}

func (p *Pool) TransferTo(op *Pool, value state.Balance) error {
	if value > p.Balance {
		return common.NewError("pool transfer failed", "value exceeds balance")
	}
	op.Balance += value
	p.Balance -= value
	return nil
}

func (p *Pool) EmptyPool() {
	p.Balance = 0
}
