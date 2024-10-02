package minersc

import (
	"encoding/json"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

//go:generate msgp -io=false -tests=false -v

// swagger:model MinerNodes
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

func (mn *MinerNodes) RemoveNodes(ids []string) {
	for _, id := range ids {
		for i, minerNode := range mn.Nodes {
			if minerNode.ID == id {
				mn.Nodes[i] = mn.Nodes[len(mn.Nodes)-1]
				mn.Nodes = mn.Nodes[:len(mn.Nodes)-1]
			}
		}
	}
}
