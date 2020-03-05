package storagesc

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/chain"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	ALL_BLOBBERS_KEY    = datastore.Key(ADDRESS + encryption.Hash("all_blobbers"))
	ALL_VALIDATORS_KEY  = datastore.Key(ADDRESS + encryption.Hash("all_validators"))
	ALL_ALLOCATIONS_KEY = datastore.Key(ADDRESS + encryption.Hash("all_allocations"))
	STORAGE_STATS_KEY   = datastore.Key(ADDRESS + encryption.Hash("all_storage"))
)

type ClientAllocation struct {
	ClientID    string       `json:"client_id"`
	Allocations *Allocations `json:"allocations"`
}

func (sn *ClientAllocation) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ClientID)
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

func (sn *ClientAllocation) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ClientAllocation) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

type Allocations struct {
	List []string
}

// func (an *Allocations) Get(idx int) string {
// 	return an[idx]
// }

func (an *Allocations) Encode() []byte {
	buff, _ := json.Marshal(an)
	return buff
}

func (an *Allocations) Decode(input []byte) error {
	err := json.Unmarshal(input, an)
	if err != nil {
		return err
	}
	return nil
}

func (an *Allocations) GetHash() string {
	return util.ToHex(an.GetHashBytes())
}

func (an *Allocations) GetHashBytes() []byte {
	return encryption.RawHash(an.Encode())
}

type ChallengeResponse struct {
	ID                string              `json:"challenge_id"`
	ValidationTickets []*ValidationTicket `json:"validation_tickets"`
}

type BlobberChallenge struct {
	BlobberID                string                       `json:"blobber_id"`
	Challenges               []*StorageChallenge          `json:"challenges"`
	ChallengeMap             map[string]*StorageChallenge `json:"-"`
	LatestCompletedChallenge *StorageChallenge            `json:"lastest_completed_challenge"`
}

func (sn *BlobberChallenge) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.BlobberID)
}

func (sn *BlobberChallenge) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *BlobberChallenge) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *BlobberChallenge) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
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

func (sn *BlobberChallenge) addChallenge(challenge *StorageChallenge) bool {
	if sn.Challenges == nil {
		sn.Challenges = make([]*StorageChallenge, 0)
		sn.ChallengeMap = make(map[string]*StorageChallenge)
	}
	if _, ok := sn.ChallengeMap[challenge.ID]; !ok {
		if len(sn.Challenges) > 0 {
			lastChallenge := sn.Challenges[len(sn.Challenges)-1]
			challenge.PrevID = lastChallenge.ID
		} else if sn.LatestCompletedChallenge != nil {
			challenge.PrevID = sn.LatestCompletedChallenge.ID
		}
		sn.Challenges = append(sn.Challenges, challenge)
		sn.ChallengeMap[challenge.ID] = challenge
		return true
	}
	return false
}

type StorageChallenge struct {
	Created        common.Timestamp   `json:"created"`
	ID             string             `json:"id"`
	PrevID         string             `json:"prev_id"`
	Validators     []*ValidationNode  `json:"validators"`
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

func (sn *ValidationNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + "validator:" + sn.ID)
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

func (sn *ValidationNode) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ValidationNode) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

type ValidatorNodes struct {
	Nodes []*ValidationNode
}

func (sn *ValidatorNodes) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *ValidatorNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *ValidatorNodes) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *ValidatorNodes) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
}

// Terms represents Blobber terms. A Blobber can update its terms,
// but any existing offer will use terms of offer signing time.
type Terms struct {
	// ReadPrice is price for reading. Token / GB.
	ReadPrice int64 `json:"read_price"`
	// WritePrice is price for reading. Token / GB. Also,
	// it used to calculate min_lock_demand value.
	WritePrice int64 `json:"write_price"`
	// MinLockDemand in number in [0; 1] range. It represents part of
	// allocation should be locked for the blobber rewards even if
	// user never write something to the blobber.
	MinLockDemand float64 `json:"min_lock_demand"`
	// MaxOfferDuration with this prices and the demand.
	MaxOfferDuration time.Duration `json:"max_offer_duration"`
	// ChallengeCompletionTime is duration required to complete a challenge.
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
}

// validate a received terms
func (t *Terms) validate() (err error) {
	if t.ReadPrice < 0 {
		return errors.New("negative read_price")
	}
	if t.WritePrice < 0 {
		return errors.New("negative write_price")
	}
	if t.MinLockDemand < 0.0 || t.MinLockDemand > 1.0 {
		return errors.New("invalid min_lock_demand")
	}
	// TODO (sfxdx): add min offer time to configurations
	// (temporary value used for development)
	if t.MaxOfferDuration < 10*time.Minute {
		return errors.New("insufficient max_offer_duration")
	}
	if t.ChallengeCompletionTime < 0 {
		return errors.New("negative challenge_completion_time")
	}
	return // nil
}

// StorageNode represents Blobber configurations.
type StorageNode struct {
	ID        string `json:"id"`
	BaseURL   string `json:"url"`
	Terms     Terms  `json:"terms"`    // terms
	Capacity  int64  `json:"capacity"` // total blobber capacity
	CapUsed   int64  `json:"cap_used"` // allocated capacity for this time
	PublicKey string `json:"-"`
}

// validate the blobber configurations
func (sn *StorageNode) validate() (err error) {
	if err = sn.Terms.validate(); err != nil {
		return
	}
	// TODO (sfxdx): add min offer time to configurations
	// (temporary value used for development, 1MB)
	if sn.Capacity <= 1*1024*1024 {
		return errors.New("insufficient blobber capacity")
	}
	return
}

func (sn *StorageNode) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ID)
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

type StorageNodes struct {
	Nodes []*StorageNode
}

func (sn *StorageNodes) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *StorageNodes) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}

func (sn *StorageNodes) GetHash() string {
	return util.ToHex(sn.GetHashBytes())
}

func (sn *StorageNodes) GetHashBytes() []byte {
	return encryption.RawHash(sn.Encode())
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
	// Terms of the Blobber at the time of signing the offer.
	Terms Terms `json:"terms"`
	// MinLockDemand for the allocation in tokens.
	MinLockDemand int64 `json:"min_lock_demand"`
}

// PriceRange represents a price range allowed by user to filter blobbers.
type PriceRange struct {
	Min int64 `json:"min"`
	Max int64 `json:"max"`
}

// isValid price range.
func (pr *PriceRange) isValid() bool {
	return 0 <= pr.Min && pr.Min <= pr.Max
}

// isMatch given price
func (pr *PriceRange) isMatch(price int64) bool {
	return pr.Min <= price && price <= pr.Max
}

// StorageAllocation request and entity.
type StorageAllocation struct {
	ID                string                        `json:"id"`
	DataShards        int                           `json:"data_shards"`
	ParityShards      int                           `json:"parity_shards"`
	Size              int64                         `json:"size"`
	Expiration        common.Timestamp              `json:"expiration_date"`
	Blobbers          []*StorageNode                `json:"blobbers"`
	Owner             string                        `json:"owner_id"`
	OwnerPublicKey    string                        `json:"owner_public_key"`
	Payer             string                        `json:"payer_id"`
	Stats             *StorageAllocationStats       `json:"stats"`
	PreferredBlobbers []string                      `json:"preferred_blobbers"`
	BlobberDetails    []*BlobberAllocation          `json:"blobber_details"`
	BlobberMap        map[string]*BlobberAllocation `json:"-"`
	ReadPriceRange    PriceRange                    `json:"read_price_range"`
	WritePriceRange   PriceRange                    `json:"write_price_range"`
	// MinLockDemand represents number of tokens required by
	// blobbers to create physical allocation.
	MinLockDemand int64 `json:"min_lock_demand"`
	// ChallengeCompletionTime is max challenge completion time of
	// all blobbers of the allocation.
	ChallengeCompletionTime time.Duration `json:"challenge_completion_time"`
}

func (sa *StorageAllocation) validate() (err error) {
	if !sa.ReadPriceRange.isValid() {
		return errors.New("invalid read_price range")
	}
	if !sa.WritePriceRange.isValid() {
		return errors.New("invalid write price range")
	}
	// TODO (sfxdx): make the min possible size configurable for sc
	// (temporary use hardcoded stub, 1MB)
	if sa.Size < 1*1024*1024 {
		return errors.New("insufficient allocation size")
	}
	var dur = common.ToTime(sa.Expiration).Sub(time.Now())
	// TODO (sfxdx): add min allocation duration to configurations
	// (temporary value used for development)
	if dur < 10*time.Minute {
		return errors.New("insufficient allocation duration")
	}

	if sa.DataShards <= 0 {
		return errors.New("invalid number of data shards")
	}

	if sa.OwnerPublicKey == "" {
		return errors.New("missing owner public key")
	}

	if sa.Owner == "" {
		return errors.New("missing owner id")
	}

	if sa.Payer == "" {
		return errors.New("missing payer id")
	}

	return // nil
}

func (sa *StorageAllocation) filterBlobbers(list []*StorageNode, bsize int64) (
	filtered []*StorageNode) {

	var (
		dur = common.ToTime(sa.Expiration).Sub(time.Now())
		i   int
	)
	for _, b := range list {
		// filter by max offer duration
		if b.Terms.MaxOfferDuration < dur {
			continue
		}
		// filter by read price
		if !sa.ReadPriceRange.isMatch(b.Terms.ReadPrice) {
			continue
		}
		// filter by write price
		if !sa.WritePriceRange.isMatch(b.Terms.WritePrice) {
			continue
		}
		// filter by blobber's capacity left
		if b.Capacity-b.CapUsed < bsize {
			continue
		}
		list[i] = b
		i++
	}
	return list[:i]
}

func (sn *StorageAllocation) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + sn.ID)
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
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(clientPublicKey)
	sigOK, err := signatureScheme.Verify(wm.Signature, signatureHash)
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

func (rc *ReadConnection) GetKey(globalKey string) datastore.Key {
	return datastore.Key(globalKey + encryption.Hash(rc.ReadMarker.BlobberID+":"+rc.ReadMarker.ClientID))
}

func (rc *ReadConnection) Decode(input []byte) error {
	err := json.Unmarshal(input, rc)
	if err != nil {
		return err
	}
	return nil
}

func (rc *ReadConnection) Encode() []byte {
	buff, _ := json.Marshal(rc)
	return buff
}

func (rc *ReadConnection) GetHash() string {
	return util.ToHex(rc.GetHashBytes())
}

func (rc *ReadConnection) GetHashBytes() []byte {
	return encryption.RawHash(rc.Encode())
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
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(clientPublicKey)
	sigOK, err := signatureScheme.Verify(rm.Signature, signatureHash)
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
	signatureScheme := chain.GetServerChain().GetSignatureScheme()
	signatureScheme.SetPublicKey(vt.ValidatorKey)
	verified, err := signatureScheme.Verify(vt.Signature, hash)
	return verified, err
}

type StorageStats struct {
	Stats              *StorageAllocationStats `json:"stats"`
	LastChallengedSize int64                   `json:"last_challenged_size"`
	LastChallengedTime common.Timestamp        `json:"last_challenged_time"`
}

func (sn *StorageStats) GetKey(globalKey string) datastore.Key {
	return STORAGE_STATS_KEY
}

func (sn *StorageStats) Decode(input []byte) error {
	err := json.Unmarshal(input, sn)
	if err != nil {
		return err
	}
	return nil
}

func (sn *StorageStats) Encode() []byte {
	buff, _ := json.Marshal(sn)
	return buff
}
