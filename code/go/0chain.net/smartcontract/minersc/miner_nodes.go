package minersc

import (
	"encoding/json"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//go:generate msgp -io=false -tests=false -v

type MinerNodes struct {
	Nodes []*MinerNode
}

func (mn *MinerNodes) Encode() []byte {
	buff, _ := json.Marshal(mn)
	return buff
}

func (mn *MinerNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, mn)
	if err != nil {
		return err
	}
	return nil
}

func (mn *MinerNodes) GetHash() string {
	return util.ToHex(mn.GetHashBytes())
}

func (mn *MinerNodes) GetHashBytes() []byte {
	return encryption.RawHash(mn.Encode())
}

func (mn *MinerNodes) FindNodeById(id string) *MinerNode {
	for _, minerNode := range mn.Nodes {
		if minerNode.ID == id {
			return minerNode
		}
	}
	return nil
}
