package feesc

import (
	"encoding/json"

	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/core/datastore"
)

type globalNode struct {
	ID        datastore.Key
	LastRound int64
}

func (gn *globalNode) encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) decode(input []byte) error {
	return json.Unmarshal(input, gn)
}

func (gn *globalNode) getKey() smartcontractstate.Key {
	return smartcontractstate.Key("fee_sc" + Seperator + gn.ID)
}
