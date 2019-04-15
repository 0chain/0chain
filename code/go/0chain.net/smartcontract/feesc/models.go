package feesc

import (
	"encoding/json"

	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

type globalNode struct {
	ID        datastore.Key
	LastRound int64
}

func (gn *globalNode) Encode() []byte {
	buff, _ := json.Marshal(gn)
	return buff
}

func (gn *globalNode) Decode(input []byte) error {
	return json.Unmarshal(input, gn)
}

func (gn *globalNode) GetKey() datastore.Key {
	return datastore.Key(gn.ID + gn.ID)
}

func (gn *globalNode) GetHash() string {
	return util.ToHex(gn.GetHashBytes())
}

func (gn *globalNode) GetHashBytes() []byte {
	return encryption.RawHash(gn.Encode())
}
