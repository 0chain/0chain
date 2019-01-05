package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/smartcontractstate"
	"0chain.net/state"
)

type LimitRequest struct {
	Pour_limit       state.Balance `json:"pour_limit"`
	Periodic_limit   state.Balance `json:"periodic_limit"`
	Global_limit     state.Balance `json:"global_limit"`
	Individual_reset time.Duration `json:"individual_reset"` //in hours
	Global_rest      time.Duration `json:"global_rest"`      //in hours
}

func (lr *LimitRequest) Encode() []byte {
	buff, _ := json.Marshal(lr)
	return buff
}

func (lr *LimitRequest) Decode(input []byte) error {
	err := json.Unmarshal(input, lr)
	return err
}

type PeriodicResponse struct {
	Used    state.Balance `json:"tokens_poured"`
	Start   time.Time     `json:"start_time"`
	Restart string        `json:"time_left"`
	Allowed state.Balance `json:"tokens_allowed"`
}

func (pr *PeriodicResponse) Encode() []byte {
	buff, _ := json.Marshal(pr)
	return buff
}

func (pr *PeriodicResponse) Decode(input []byte) error {
	err := json.Unmarshal(input, pr)
	return err
}

type GlobalNode struct {
	ID               string        `json:"id"`
	Pour_limit       state.Balance `json:"pour_limit"`
	Periodic_limit   state.Balance `json:"periodic_limit"`
	Global_limit     state.Balance `json:"global_limit"`
	Individual_reset string        `json:"individual_reset"` //in hours
	Global_reset     string        `json:"global_rest"`      //in hours
	Used             state.Balance `json:"used"`
	StartTime        time.Time     `json:"start_time"`
}

func (gn *GlobalNode) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("faucet_contract:" + gn.ID)
}

func (gn *GlobalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *GlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

type UserNode struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	Used      state.Balance `json:"used"`
}

func (un *UserNode) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("faucet_user:" + un.ID)
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}
