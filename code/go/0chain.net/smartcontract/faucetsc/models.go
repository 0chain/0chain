package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/chaincore/state"
)

type limitRequest struct {
	PourAmount      state.Balance `json:"pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
}

func (lr *limitRequest) encode() []byte {
	buff, _ := json.Marshal(lr)
	return buff
}

func (lr *limitRequest) decode(input []byte) error {
	err := json.Unmarshal(input, lr)
	return err
}

type periodicResponse struct {
	Used    state.Balance `json:"tokens_poured"`
	Start   time.Time     `json:"start_time"`
	Restart string        `json:"time_left"`
	Allowed state.Balance `json:"tokens_allowed"`
}

func (pr *periodicResponse) encode() []byte {
	buff, _ := json.Marshal(pr)
	return buff
}

func (pr *periodicResponse) decode(input []byte) error {
	err := json.Unmarshal(input, pr)
	return err
}

type globalNode struct {
	ID              string        `json:"id"`
	PourAmount      state.Balance `json:"pour_amount"`
	PeriodicLimit   state.Balance `json:"periodic_limit"`
	GlobalLimit     state.Balance `json:"global_limit"`
	IndividualReset time.Duration `json:"individual_reset"` //in hours
	GlobalReset     time.Duration `json:"global_rest"`      //in hours
	Used            state.Balance `json:"used"`
	StartTime       time.Time     `json:"start_time"`
}

func (gn *globalNode) getKey() smartcontractstate.Key {
	return smartcontractstate.Key("faucet_contract:" + gn.ID)
}

func (gn *globalNode) encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) decode(input []byte) error {
	err := json.Unmarshal(input, gn)
	return err
}

type userNode struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	Used      state.Balance `json:"used"`
}

func (un *userNode) getKey() smartcontractstate.Key {
	return smartcontractstate.Key("faucet_user:" + un.ID)
}

func (un *userNode) encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *userNode) decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}
