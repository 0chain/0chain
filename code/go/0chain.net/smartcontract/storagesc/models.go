package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/chaincore/smartcontractstate"
	"0chain.net/core/common"
	"0chain.net/core/encryption"
)

var ALL_BLOBBERS_KEY = smartcontractstate.Key("all_blobbers")
var ALL_VALIDATORS_KEY = smartcontractstate.Key("all_validators")
var ALL_ALLOCATIONS_KEY = smartcontractstate.Key("all_allocations")

type ClientAllocation struct {
	ClientID    string   `json:"client_id"`
	Allocations []string `json:"allocations"`
}

func (sn *ClientAllocation) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("client:" + sn.ClientID)
}

func (sn *ClientAllocation) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ClientAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type StorageChallenge struct {
	ID             string             `json:"id"`
	Validators     []ValidationNode   `json:"validators"`
	RandomNumber   int64              `json:"seed"`
	AllocationID   string             `json:"allocation_id"`
	Blobber        *StorageNode       `json:"blobber"`
	AllocationRoot string             `json:"allocation_root"`
	Response       *ChallengeResponse `json:"challenge_response"`
}

func (sn *StorageChallenge) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("challenge:" + sn.ID)
}

func (sn *StorageChallenge) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageChallenge) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

type ValidationNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	PublicKey string `json:"-"`
}

func (sn *ValidationNode) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("validator:" + sn.ID)
}

func (sn *ValidationNode) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ValidationNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

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
	ID             string           `json:"id"`
	DataShards     int              `json:"data_shards"`
	ParityShards   int              `json:"parity_shards"`
	Size           int64            `json:"size"`
	UsedSize       int64            `json:"used_size"`
	Expiration     common.Timestamp `json:"expiration_date"`
	Blobbers       []*StorageNode   `json:"blobbers"`
	Owner          string           `json:"owner_id"`
	OwnerPublicKey string           `json:"owner_public_key"`
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

type BlobberCloseConnection struct {
	AllocationRoot     string       `json:"allocation_root"`
	PrevAllocationRoot string       `json:"prev_allocation_root"`
	WriteMarker        *WriteMarker `json:"write_marker"`
}

func (bc *BlobberCloseConnection) Decode(input []byte) error {
	err := json.Unmarshal(input, bc)
	if err != nil {
		return err
	}
	return nil
}

func (bc *BlobberCloseConnection) Verify() bool {
	if bc.WriteMarker == nil {
		return false
	}
	if len(bc.AllocationRoot) == 0 {
		return false
	}

	if bc.WriteMarker.AllocationRoot != bc.AllocationRoot {
		//return "", common.NewError("invalid_parameters", "Invalid Allocation root. Allocation root in write marker does not match the commit")
		return false
	}

	if bc.WriteMarker.PreviousAllocationRoot != bc.PrevAllocationRoot {
		//return "", common.NewError("invalid_parameters", "Invalid Previous Allocation root. Previous Allocation root in write marker does not match the commit")
		return false
	}
	return bc.WriteMarker.Verify()

}

type WriteMarker struct {
	AllocationRoot         string           `json:"allocation_root"`
	PreviousAllocationRoot string           `json:"prev_allocation_root"`
	AllocationID           string           `json:"allocation_id"`
	Size                   int64            `json:"size"`
	BlobberID              string           `json:"blobber_id"`
	Timestamp              common.Timestamp `json:"timestamp"`
	ClientID               string           `json:"client_id"`
	Signature              string           `json:"signature"`
}

func (wm *WriteMarker) VerifySignature(clientPublicKey string) bool {
	hashData := wm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	sigOK, err := encryption.Verify(clientPublicKey, wm.Signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (wm *WriteMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", wm.AllocationRoot, wm.PreviousAllocationRoot, wm.AllocationID, wm.BlobberID, wm.ClientID, wm.Size, wm.Timestamp)
	return hashData
}

func (wm *WriteMarker) Verify() bool {
	if len(wm.AllocationID) == 0 || len(wm.AllocationRoot) == 0 || len(wm.BlobberID) == 0 || len(wm.ClientID) == 0 || wm.Timestamp == 0 {
		return false
	}
	return true
}

type ReadConnection struct {
	ReadMarker *ReadMarker `json:"read_marker"`
}

func (rc *ReadConnection) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("rm:" + encryption.Hash(rc.ReadMarker.BlobberID+":"+rc.ReadMarker.ClientID))
}

func (rc *ReadConnection) Decode(input []byte) error {
	err := json.Unmarshal(input, rc)
	if err != nil {
		return err
	}
	return nil
}

type ReadMarker struct {
	ClientID        string           `json:"client_id"`
	ClientPublicKey string           `json:"client_public_key"`
	BlobberID       string           `json:"blobber_id"`
	AllocationID    string           `json:"allocation_id"`
	OwnerID         string           `json:"owner_id"`
	Timestamp       common.Timestamp `json:"timestamp"`
	ReadCounter     int64            `json:"counter"`
	Signature       string           `json:"signature"`
}

func (rm *ReadMarker) VerifySignature(clientPublicKey string) bool {
	hashData := rm.GetHashData()
	signatureHash := encryption.Hash(hashData)
	sigOK, err := encryption.Verify(clientPublicKey, rm.Signature, signatureHash)
	if err != nil {
		return false
	}
	if !sigOK {
		return false
	}
	return true
}

func (rm *ReadMarker) GetHashData() string {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v:%v", rm.AllocationID, rm.BlobberID, rm.ClientID, rm.ClientPublicKey, rm.OwnerID, rm.ReadCounter, rm.Timestamp)
	return hashData
}

func (rm *ReadMarker) Verify(prevRM *ReadMarker) bool {
	if len(rm.AllocationID) == 0 || rm.ReadCounter <= 0 || len(rm.BlobberID) == 0 || len(rm.ClientID) == 0 || rm.Timestamp == 0 || len(rm.OwnerID) == 0 {
		return false
	}
	if prevRM != nil {
		if rm.BlobberID != prevRM.BlobberID || rm.OwnerID != prevRM.OwnerID || rm.Timestamp <= prevRM.Timestamp || rm.ReadCounter <= prevRM.ReadCounter {
			return false
		}
	}

	return rm.VerifySignature(rm.ClientPublicKey)
}

type ValidationTicket struct {
	ChallengeID  string           `json:"challenge_id"`
	BlobberID    string           `json:"blobber_id"`
	ValidatorID  string           `json:"validator_id"`
	ValidatorKey string           `json:"validator_key"`
	Result       bool             `json:"success"`
	Message      string           `json:"message"`
	MessageCode  string           `json:"message_code"`
	Timestamp    common.Timestamp `json:"timestamp"`
	Signature    string           `json:"signature"`
}

func (vt *ValidationTicket) VerifySign() (bool, error) {
	hashData := fmt.Sprintf("%v:%v:%v:%v:%v:%v", vt.ChallengeID, vt.BlobberID, vt.ValidatorID, vt.ValidatorKey, vt.Result, vt.Timestamp)
	hash := encryption.Hash(hashData)
	verified, err := encryption.Verify(vt.ValidatorKey, vt.Signature, hash)
	return verified, err
}
