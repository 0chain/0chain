package faucetsc

import (
	"encoding/json"
	"time"

	"0chain.net/chaincore/state"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type limitRequest struct {
	PourAmount      state.Balance `json:"pour_amount"`
	MaxPourAmount   state.Balance `json:"max_pour_amount"`
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

type UserNode struct {
	ID        string        `json:"id"`
	StartTime time.Time     `json:"start_time"`
	Used      state.Balance `json:"used"`
}

func (un *UserNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + un.ID)
}

func (un *UserNode) GetHash() string {
	return util.ToHex(un.GetHashBytes())
}

func (un *UserNode) GetHashBytes() []byte {
	return encryption.RawHash(un.Encode())
}

func (un *UserNode) Encode() []byte {
	buff, _ := json.Marshal(un)
	return buff
}

func (un *UserNode) Decode(input []byte) error {
	err := json.Unmarshal(input, un)
	return err
}
