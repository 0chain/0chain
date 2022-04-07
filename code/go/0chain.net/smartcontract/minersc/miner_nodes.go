package minersc

import (
	"encoding/json"

	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

//msgp:ignore MinerNodes
//go:generate msgp -io=false -tests=false -v

// swagger:model MinerNodes
type MinerNodes struct {
	Nodes []*MinerNode
}

type VCPoolLockNodes struct {
	Nodes []*minerNodeDecode
}

func (mn *MinerNodes) MarshalMsg(o []byte) ([]byte, error) {
	nn := &VCPoolLockNodes{
		Nodes: make([]*minerNodeDecode, len(mn.Nodes)),
	}

	for i, n := range mn.Nodes {
		nn.Nodes[i] = newDecodeFromMinerNode(n)
	}

	return nn.MarshalMsg(o)
}

func (mn *MinerNodes) UnmarshalMsg(data []byte) ([]byte, error) {
	nn := &VCPoolLockNodes{}
	o, err := nn.UnmarshalMsg(data)
	if err != nil {
		return nil, err
	}

	ns := &MinerNodes{
		Nodes: make([]*MinerNode, len(nn.Nodes)),
	}

	for i, n := range nn.Nodes {
		ns.Nodes[i] = n.toMinerNode()
	}

	*mn = *ns
	return o, nil
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
