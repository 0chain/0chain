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

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type BlobberChallenge struct {
	BlobberID                 string                       `json:"blobber_id"`
	Challenges                []*StorageChallenge          `json:"challenges"`
	ChallengeMap              map[string]*StorageChallenge `json:"-"`
	LatestCompletedChallenges []*StorageChallenge          `json:"lastest_completed_challenges"`
}

func (sn *BlobberChallenge) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("blobber_challenge:" + sn.BlobberID)
}

func (sn *BlobberChallenge) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *BlobberChallenge) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	sn.ChallengeMap = make(map[string]*StorageChallenge)
	for _, challenge := range sn.Challenges {
		sn.ChallengeMap[challenge.ID] = challenge
	}
	return nil
}

func (sn *BlobberChallenge) addChallenge(challenge *StorageChallenge) {
	if sn.Challenges == nil {
		sn.Challenges = make([]*StorageChallenge, 0)
		sn.ChallengeMap = make(map[string]*StorageChallenge)
	}
	sn.Challenges = append(sn.Challenges, challenge)
	sn.ChallengeMap[challenge.ID] = challenge
}

type StorageChallenge struct {
	Created        common.Timestamp   `json:"created"`
	ID             string             `json:"id"`
	Validators     []ValidationNode   `json:"validators"`
	RandomNumber   int64              `json:"seed"`
	AllocationID   string             `json:"allocation_id"`
	Blobber        *StorageNode       `json:"blobber"`
	AllocationRoot string             `json:"allocation_root"`
	Response       *ChallengeResponse `json:"challenge_response,omitempty"`
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

type StorageAllocationStats struct {
	UsedSize                  int64  `json:"used_size"`
	NumWrites                 int64  `json:"num_of_writes"`
	NumReads                  int64  `json:"num_of_reads"`
	TotalChallenges           int64  `json:"total_challenges"`
	OpenChallenges            int64  `json:"num_open_challenges"`
	SuccessChallenges         int64  `json:"num_success_challenges"`
	FailedChallenges          int64  `json:"num_failed_challenges"`
	LastestClosedChallengeTxn string `json:"latest_closed_challenge"`
}

type BlobberAllocation struct {
	BlobberID       string                  `json:"blobber_id"`
	AllocationID    string                  `json:"allocation_id"`
	Size            int64                   `json:"size"`
	AllocationRoot  string                  `json:"allocation_root"`
	LastWriteMarker *WriteMarker            `json:"write_marker"`
	Stats           *StorageAllocationStats `json:"stats"`
}

type StorageAllocation struct {
	ID             string                        `json:"id"`
	DataShards     int                           `json:"data_shards"`
	ParityShards   int                           `json:"parity_shards"`
	Size           int64                         `json:"size"`
	Expiration     common.Timestamp              `json:"expiration_date"`
	Blobbers       []*StorageNode                `json:"blobbers"`
	Owner          string                        `json:"owner_id"`
	OwnerPublicKey string                        `json:"owner_public_key"`
	Stats          *StorageAllocationStats       `json:"stats"`
	BlobberDetails []*BlobberAllocation          `json:blobber_details`
	BlobberMap     map[string]*BlobberAllocation `json:"-"`
}

func (sn *StorageAllocation) GetKey() smartcontractstate.Key {
	return smartcontractstate.Key("allocation:" + sn.ID)
}

func (sn *StorageAllocation) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	sn.BlobberMap = make(map[string]*BlobberAllocation)
	for _, blobberAllocation := range sn.BlobberDetails {
		sn.BlobberMap[blobberAllocation.BlobberID] = blobberAllocation
	}
	return nil
}

func (sn *StorageAllocation) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
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

func (rm *ReadMarker) Verify(prevRM *ReadMarker) error {
	if rm.ReadCounter <= 0 || len(rm.BlobberID) == 0 || len(rm.ClientID) == 0 || rm.Timestamp == 0 {
		return common.NewError("invalid_read_marker", "length validations of fields failed")
	}
	if prevRM != nil {
		if rm.ClientID != prevRM.ClientID || rm.BlobberID != prevRM.BlobberID || rm.Timestamp < prevRM.Timestamp || rm.ReadCounter < prevRM.ReadCounter {
			return common.NewError("invalid_read_marker", "validations with previous marker failed.")
		}
	}
	ok := rm.VerifySignature(rm.ClientPublicKey)
	if ok {
		return nil
	}
	return common.NewError("invalid_read_marker", "Signature verification failed for the read marker")
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
