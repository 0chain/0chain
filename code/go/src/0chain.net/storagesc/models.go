package storagesc

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/encryption"
	"0chain.net/smartcontractstate"
)

var ALL_BLOBBERS_KEY = smartcontractstate.Key("all_blobbers")

type StorageNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
}

func (sn *StorageNode) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("blobber:" + sn.ID)
}

func (sn *StorageNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type StorageAllocation struct {
	ID           string           `json:"id"`
	DataShards   int              `json:"data_shards"`
	ParityShards int              `json:"parity_shards"`
	Size         int64            `json:"size"`
	UsedSize     int64            `json:"used_size"`
	Expiration   common.Timestamp `json:"expiration_date"`
	Blobbers     []*StorageNode   `json:"blobbers"`
	Owner        string           `json:"owner_id"`
}

func (sn *StorageAllocation) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("allocation:" + sn.ID)
}

func (sn *StorageAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *StorageAllocation) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

type BlobberAllocation struct {
	ID              string       `json:"id"`
	AllocationID    string       `json:"allocation_id"`
	Size            int64        `json:"size"`
	UsedSize        int64        `json:"used_size"`
	AllocationRoot  string       `json:"allocation_root"`
	BlobberID       string       `json:"blobber_id"`
	LastWriteMarker *WriteMarker `json:"write_marker"`
}

func (ba *BlobberAllocation) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("blobber_allocation:" + encryption.Hash(ba.AllocationID+":"+ba.BlobberID))
}

func (ba *BlobberAllocation) Encode() []byte {
	buff, _ := json.Marshal(ba)
	return buff
}

func (ba *BlobberAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, ba)
	if err != nil {
		return err
	}
	return nil
}
